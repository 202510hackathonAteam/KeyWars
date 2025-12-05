# バックエンド環境

## 運用・基本方針
### ガイドライン
- 変更は既存の流れを尊重し、まずは追加で拡張する方針です。  
  大きな変更が必要な場合は、簡単な提案（理由と影響範囲）を共有いただけると助かります。
- 依存関係の組み立ては `internal/app/app.go` に集約しています。  
  通常は各所で直接 `NewXxx()` を呼ばず、`app.go` で配線します。

### コードスタイル
- package 名はディレクトリ名に合わせる（1 ディレクトリ＝1 package）
  例）`internal/service/auth_service.go` → `package service`
- 命名は複数形のエンドポイントに合わせる（例：`/matches` → `MatchesHandler`）。

<br>

## 新しいService／Repository／Handlerを追加する方法
### 1. Serviceを追加する
1. `internal/service/` に、`<feature>_service.go` を新規作成。  
    新しいサービスは、以下のようにインターフェース／実装構造体／コンストラクタ関数を最低限用意します。  
    ```go
    package service

    import "keywars/backend/internal/domain/repository"

    // XxxService は <機能> のユースケースを定義するインターフェース。
    type XxxService interface {
      // 例: CreateXxx(ctx context.Context, input XxxInput) (XxxOutput, error)
    }

    // xxxService は XxxService の具象実装。
    type xxxService struct {
      xxx repository.XxxRepository
    }

    // NewXxxService は XxxService 実装を生成。
    func NewXxxService(repo repository.XxxRepository) XxxService {
      return &xxxService{xxx: repo}
    }
    ```
2. `internal/service/module.go` に追加。
    ```go
    var Module = fx.Module(
      fx.Provide(
        NewXxxService, // ← 追加
      ),
    )
    ```

### 2. Repository を追加する
1. Domain の Repository Interface を追加。
`internal/domain/repository/xxx_repository.go` に必要なリポジトリインターフェースを追加（存在しない場合）。  
    ```go
    package repository

    // XxxRepository は <機能> の永続化操作を定義するインターフェース。
    type XxxRepository interface{}
    ```
2. SQL 実装を追加。
もし SQL を使う場合は、`internal/infra/repository/sql/xxx_repository.go` に以下を作成。
    ```go
    package sql

    import (
      "gorm.io/gorm"
      "keywars/backend/internal/domain/repository"
    )

    // xxxRepositorySQL は XxxRepository の SQL 実装。
    type xxxRepo struct {
      db *gorm.DB
    }

    // NewXxxRepositorySQL は SQL リポジトリを生成。
    func NewXxxRepo(db *gorm.DB) repository.XxxRepository {
      return &xxxRepo{db: db}
    }
    ```
3. Fx Module に Repository を登録。
`internal/infra/repository/sql/module.go` に追加。
    ```go
    var Module = fx.Module(
      fx.Provide(
        NewXxxRepositorySQL, // ← 追加
      ),
    )
    ```

### 3. Handler を追加する
1. Handler を作成。
`internal/transport/http/handler/<feature>s_handler.go` に `Handler` を追加。
    ```go
    package handler

    import "keywars/backend/internal/service"

    // XxxsHandler は <機能> の HTTP Handler。
    type XxxsHandler struct {
      xxxService service.XxxService
    }

    // NewXxxsHandler は Handler を生成。
    func NewXxxsHandler(s service.XxxService) *XxxsHandler {
      return &XxxsHandler{xxxService: s}
    }
    ```
2. Fx Module に Handler を登録。
`internal/transport/http/handler/module.go`に追加。
    ```go
    var Module = fx.Module(
      "handler",
      fx.Provide(
        NewXxxsHandler, // ← 追加
      ),
    )
    ```

### 4. Router にハンドラーを紐付ける
1. Fx では Router Module でハンドラーを受け取り、ルーティング登録。
`router/router.go` に認証不要のAPI、認証必須のAPIグループどちらかのルートを登録。
    ```go
    // Handlers は、ルーターで利用する HTTP ハンドラー群を Fx から受け取るための依存セット。
    type Handlers struct {
      fx.In
      Auth *handler.AuthHandler
      Matches *handler.MatchHandler // ← 新しいハンドラーを追加
    }

    // ──────────────────────────────
    // 認証不要のAPI
    // ──────────────────────────────
    e.GET("/matches", handlers.Matches.GetMatches) // ← どちらかに追加

    // ──────────────────────────────
    // 認証必須のAPIグループ (/api/v1)
    // ──────────────────────────────
    v1 := e.Group("/api/v1", authMiddleware)
    v1.GET("/matches", handlers.Matches.GetMatches) // ← どちらかに追加
    ```
### 5. di 統合は Fx が自動で実行
`internal/di/module.go`:  

  ```go
  var Module = fx.Options(
    config.Module,
    db.Module,
    redisx.Module,

    // SQL repos
    sqlrepository.Module,

    // Redis
    redisrepository.Module,

    // JWT
    auth.Module,

    // Service
    service.Module,
    round.Module,
    realtime.Module,

    // middleware / validator
    httpmiddleware.Module,
    validator.Module,

    // handler
    handler.Module,

    // websocket
    ws.Module,

    app.Module,
  )
  ```

`cmd/server/main.go`:  
```go
func main() {
  fx.New(di.Module).Run()
}
```

<br>  

## ハンドラー層でのログ出力とレスポンス方針
### 原則
ハンドラー層では、エラー発生時に **ログ出力とレスポンス返却（response.Respond）をセットで行います**。

サービス層・リポジトリ層ではログを出さず、`error` を返すのみとします。  
これにより「どのリクエストで何が失敗したか」が必ずハンドラー側で記録されます。  

### ログレベルの使い分け
| レベル | 用途 | 例 |
|--------|------|----|
| `Error` | サーバー内部の異常（例：DB接続失敗） | `logger.Error().Err(err).Msg("failed to connect DB")` |
| `Warn`  | 想定内の軽度な異常（例：入力不備・認証失敗） | `logger.Warn().Err(err).Msg("signin failed: user not found")` |
| `Info`  | 正常系の操作や主要イベント | `logger.Info().Str("user", id).Msg("user signed up")` |
| `Debug` | 詳細なデバッグ情報（開発時のみ有効） | `logger.Debug().Msg("token parsed successfully")` |

### 実装ルール
| ケース | ログ出力 | HTTPステータス | レスポンスメッセージ例 |
|---------|-----------|----------------|--------------------------|
| クライアント入力ミス (400系) | `Warn()` | 400 / 401 / 409 | `"invalid credentials"`, `"user already exists"` |
| サーバー内部エラー (500系) | `Error()` | 500 | `"internal error"` |
| 正常処理 | `Info()`（必要に応じて） | 200 / 201 / 204 | `"ok"`, `"created"` |

### 例外（ログ不要なケース）
- 想定内のバリデーションエラー（username is requiredなど）
- Cookie やヘッダが存在しないなど日常的な400系エラー
→ response.Respond のみでOK（ログノイズ防止）

### コーディング例
```go
import (
  "github.com/rs/zerolog"

  "keywars/backend/internal/transport/http/response"
)

logger := zerolog.Ctx(c.Request().Context())
logger.Warn().Err(err).Msg("signin failed: invalid credentials")
return response.Respond(c, http.StatusUnauthorized, echo.Map{
  "message": "invalid credentials",
})
```

<br>  

## 必須ではない環境変数とカスタマイズ方法
`.env.example` に記載がない一部の変数も、挙動を変更したい場合に利用できます。 

| 変数名 | デフォルト | 説明 |
|--------|-------------|------|
| `CORS_ALLOWED_ORIGINS` | 空 | フロントエンドなどからのリクエストを許可するオリジンを指定します。複数指定する場合はカンマ区切りで記載。 |

> 例：
> ```bash
> CORS_ALLOWED_ORIGINS=http://localhost:3000,http://127.0.0.1:3000
> ```
> ※ 指定がない場合は全てのオリジンからのリクエストが拒否されます。

### 補足
- `CORS_ALLOWED_ORIGINS` は、フロントエンドのURLを明示的に許可したい場合に設定します。  
  （未設定時はワイルドカード `*` が使用され、全てのオリジンからのアクセスを許可します）  

<br>

## ヘルスチェック
- /healthz（公開）…起動確認用  
  （例： [http://localhost:8080/healthz](http://localhost:8080/healthz) ）

<br>

## ディレクトリ構成
```text
backend/
├─ cmd/        # アプリの実行入口（server起動 / DBマイグレーション実行）
│  ├─ seed/        # 初期データ投入用の実行ディレクトリ
│  └─ server/      # サーバー起動用の実行ディレクトリ
├─ internal/   # アプリ本体（外部からはimport不可）
│  ├─ app/         # アプリ全体の初期化・依存関係の組み立て（DB→Service→Handler）
│  ├─ config/      # 環境変数・設定ファイルの読み込み（Config構造体定義）
│  ├─ di/          # 依存関係の集約ポイント（Fx Module をまとめる統合レイヤー）
│  ├─ service/     # ビジネスロジック層（アプリの振る舞い・ユースケースを記述）
│  │  ├─ realtime/   # WebSocket を用いたリアルタイム処理（入出力層）
│  │  └─ round/      # ゲーム進行ロジック（ドメインアクション。ラウンド管理の中心）
│  ├─ domain/      # データ構造と契約層（Repositoryインターフェース）
│  │  ├─ model/      # ドメインモデル定義（業務ルール中心）
│  │  ├─ port/       # アプリ内外の接続インターフェース（port定義）
│  │  ├─ repository/ # Repositoryインターフェース（契約のみを定義）
│  │  ├─ constant/   # ドメイン共通の定数（イベント種別・エラー種別など）
│  │  └─ types/      # ドメイン内で用いる基本的な型エイリアスや値オブジェクト
│  ├─ infra/       # データアクセス層（DBやRedisなど外部リソースへの実装）
│  │  ├─ auth/         # JWTなどの認証関連の実装
│  │  ├─ db/           # データベース接続の初期化や管理を担当
│  │  │  ├─ initial/   # DBマイグレーション後の初期データの投入処理を担当
│  │  │  └─ seed/      # デモ、テストデータ登録関連
│  │  ├─ redis/        # Redis接続の初期化や共通処理を担当
│  │  └─ repository/   # domainで定義したRepositoryの実装層
│  │     ├─ redis/     # Redisを用いたRepositoryの実装
│  │     │  └─ model/  # Redisに保存するデータ形式（DTO）のモデル定義
│  │     └─ sql/       # SQL(GORM)を用いたRepositoryの実装
│  │        └─ model/  # DBテーブル構造に対応するGORMモデル定義
│  ├─ transport/   # 通信層（HTTPやWebSocketでリクエストを受ける部分）
│  │  ├─ http/         # HTTP通信関連の処理をまとめる
│  │  │  ├─ handler/       # 各エンドポイントのハンドラを定義
│  │  │  ├─ middleware/    # 認証・ログなどのHTTPミドルウェアを定義
│  │  │  ├─ router/        # ルーティング設定を定義
│  │  │  └─ response/      # 共通レスポンス生成処理（HTTPレスポンスの形式統一）
│  │  └─ websocket/    # WebSocket通信関連の処理をまとめる
│  └─ util/         # 汎用的な共通処理をまとめる（アプリ全体から再利用される）
│     ├─ validator/      # 入力値の検証（バリデーション）ロジックを提供
│     └─ cookie/         # Cookie操作（設定・削除）を提供
└─ migrations/     # DBマイグレーションSQL（テーブル作成や変更）
```
