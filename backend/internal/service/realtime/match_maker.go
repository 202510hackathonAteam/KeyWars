package realtime

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"keywars/backend/internal/domain/constant"
	"keywars/backend/internal/domain/repository"
	"keywars/backend/internal/service/round"
	"keywars/backend/internal/transport/websocket"
)

//
// ==== Service 構造体 ====
//

// MatchMakerService は、マッチ待機キューを監視し、
// 一定間隔でマッチング成立を試行するバックグラウンドサービス。
//
// 責務：
//   - 待機キューからのプレイヤー取り出し
//   - 試合状態・初期データの生成
//   - 成立時の通知（接続中ユーザーのみ）
type MatchMakerService struct {
	matchQueueRepo 			 repository.MatchQueueRepository
	roundStateRepo 			 repository.RoundStateRepository
	websocketHub         *websocket.Hub
	logger 							 *zerolog.Logger
	deckGeneratorService *round.DeckGeneratorService
	roundFlowService 		 *round.RoundFlowService
}

// NewMatchMakerService は、マッチメイカーのバックグラウンド処理に必要な
// リポジトリ・Hub・ラウンド制御サービスを受け取り、
// MatchMakerService を生成するコンストラクタ。
func NewMatchMakerService(
	matchQueueRepo repository.MatchQueueRepository,
	roundStateRepo repository.RoundStateRepository,
	websocketHub *websocket.Hub,
	logger *zerolog.Logger,
	deckGeneratorService *round.DeckGeneratorService,
	roundFlowService *round.RoundFlowService,
) *MatchMakerService {
	return &MatchMakerService{
		matchQueueRepo: 	 		matchQueueRepo,
		roundStateRepo: 	 		roundStateRepo,
		websocketHub:         websocketHub,
		logger:								logger,
		deckGeneratorService: deckGeneratorService,
		roundFlowService:			roundFlowService,
	}
}

//
// ==== マッチメイカー（定期実行タスク） ====
//

// StartMatchmaker は、一定間隔でマッチング処理を試行するバックグラウンドループを開始する。
// Redis の待機キューに 2 名以上が存在する場合、Dequeue して新しいマッチを生成する。
// tick には試行間隔（例: 500ms, 1s など）を指定する。
func (s *MatchMakerService) StartMatchmaker(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// コンテキストがキャンセルされた場合はループを終了
				return

			case <-ticker.C:
				// tick ごとにマッチング処理を試行
				s.tryMakeMatch(ctx)
			}
		}
	}()
}

//
// ==== 内部マッチングロジック ====
//

// tryMakeMatch は、Redis の待機キューから 2 名を取り出し、
// 新しいマッチを初期化して各クライアントへ通知する。
// 取り出しは排他ロックにより、同時実行を防止する。
func (s *MatchMakerService) tryMakeMatch(ctx context.Context) {
	// DequeuePairAndInitMatch:
	//   - 2名を ZPOPMIN で取り出す
	//   - match:{matchID} / match:{matchID}:state を初期化
	dequeueCtx, cancelDequeue := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancelDequeue()
	user1ID, user2ID, matchID, err := s.matchQueueRepo.DequeuePairAndInitMatch(dequeueCtx)
	if err != nil || matchID == "" {
		// 競合発生 or 2名未満の場合は何もしない
		return
	}

	// 初期化
	initMeasurementCtx, cancelInitMeasurement := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancelInitMeasurement()
	if err := s.roundStateRepo.InitMeasurementFinishCount(initMeasurementCtx, matchID);err != nil {
		s.logger.Error().
			Err(err).
			Str("event", constant.EventMatchTryMake).
			Str("match_id", matchID).
			Msg("failed to initialize measurement finish count")
		return
	}

	// 問題抽出
	deckSaveCtx, cancelDeckSave := context.WithTimeout(ctx, 700*time.Millisecond)
	defer cancelDeckSave()
	if err := s.deckGeneratorService.GenerateAndSaveDeck(deckSaveCtx, matchID);err != nil {
		s.logger.Error().
			Err(err).
			Str("event", constant.EventMatchTryMake).
			Str("match_id", matchID).
			Msg("failed to generate prompts")
		return
	}

	// ユーザー → 試合の対応関係を原子的に確定(途中失敗による不整合状態を防ぐ)
	setUsersCtx, cancelSetUsers := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancelSetUsers()
	if err := s.roundStateRepo.SetActiveMatchForUsers(setUsersCtx, user1ID, user2ID, matchID); err != nil {
		s.logger.Error().
			Err(err).
			Str("event", constant.EventMatchTryMake).
			Str("match_id", matchID).
			Msg("failed to activate match for users")
		return
	}

	// マッチ成立通知（即時 push）
  s.notifyMatchFoundIfConnected(user1ID, matchID, user2ID)
  s.notifyMatchFoundIfConnected(user2ID, matchID, user1ID)

	// 第1ラウンド開始（1回だけ）
	if err := s.roundFlowService.StartFirstRound(ctx, matchID, user1ID, user2ID); err != nil {
    s.logger.Error().
			Err(err).
			Str("event", constant.EventMatchTryMake).
			Str("matchID", matchID).
			Msg("failed to start first round")
    return
	}
}

//
// ==== 内部ヘルパー ====
//

// moveClientIfConnected は、指定されたユーザーIDのクライアントが現在オンラインであれば、
// 新しいルーム（newRoomName）へ安全に移動させる。
// 切断済み（Hubに存在しない）場合は no-op（何もしない）。
func (s *MatchMakerService) moveClientIfConnected(userID string, newRoomName string) error {
	// 個人ルーム（user:<userID>）に現在接続中のクライアントを取得
	clientConnections := s.websocketHub.Members("user:" + userID)
	if len(clientConnections) == 0 {
		// 未接続または直前に切断された場合
		return nil
	}

	// 想定上、同一ユーザーに対して複数の接続は存在しない。
	// 仮に複数ある場合は、最初の1件のみを対象とする。
	s.websocketHub.Move(clientConnections[0], newRoomName)
	return nil
}

// notifyMatchFoundIfConnected は、接続中ユーザーに対するマッチ成立の即時通知処理。
// WebSocket 接続が存在する場合のみ通知とルーム移動を行い、
// 未接続の場合は OnConnect による再送に委ねる実装。
func (s *MatchMakerService) notifyMatchFoundIfConnected(
  userID, matchID, opponentID string,
) {
  conns := s.websocketHub.Members("user:" + userID)
  if len(conns) == 0 {
    return // 未接続なら後回し
  }

  if err := s.websocketHub.Move(conns[0], "match:"+matchID); err != nil {
		s.logger.Debug().
			Err(err).
			Str("match_id", matchID).
			Msg("failed to move websocket room (best-effort)")
		return
	}
  if err := conns[0].SendJSON(websocket.NewMatchFoundPayload(matchID, opponentID)); err != nil {
		s.logger.Debug().
			Err(err).
			Str("match_id", matchID).
			Msg("failed to send match found payload (best-effort)")
	}
}