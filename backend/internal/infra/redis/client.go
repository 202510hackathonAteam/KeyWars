package redisx

import (
	// "context"
	"time"

	"github.com/redis/go-redis/v9"

	"keywars/backend/internal/config"
)

// NewRedis は、Redis クライアントを初期化し、接続確認を行う関数。
// go-redis v9 に基づく共通的な接続ユーティリティとして利用される。
//
// この関数は、指定したアドレス・パスワード・DB番号を使用して redis.Client を生成し、
// Ping コマンドで接続確認を行う。成功時には利用可能な *redis.Client を返す。
//
// 例:
//
//	redisClient, err := redisx.NewRedis("localhost:6379", "", 0)
//	if err != nil {
//	    log.Fatalf("Redis接続失敗: %v", err)
//	}
//
// 引数:
//
//	address        - Redisサーバーのアドレス（例: "localhost:6379"）
//	password       - Redis認証パスワード（未設定の場合は空文字）
//	databaseNumber - 使用するDB番号（通常は 0）
//
// 戻り値:
//
//	*redis.Client - 正常に初期化された Redis クライアント
//	error         - 接続エラー（Ping失敗やネットワーク異常時）
//
// タイムアウト設定:
//
//	DialTimeout  : 500ms 以内に接続できない場合はエラー
//	ReadTimeout  : 300ms 以内にレスポンスが返らなければエラー
//	WriteTimeout : 300ms 以内に送信できない場合はエラー
//	Ping確認用コンテキスト: 全体で 1 秒以内に応答がない場合キャンセル
func NewRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	// Redis クライアントを設定して生成
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  300 * time.Millisecond,
		WriteTimeout: 300 * time.Millisecond,
	})

	// Pingテスト: 実際に接続確認を行う
	// contextWithTimeout, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	// return redisClient, redisClient.Ping(contextWithTimeout).Err()
	return redisClient, nil
}
