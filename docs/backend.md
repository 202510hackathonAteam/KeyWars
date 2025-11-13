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

## 新しいService／Handlerを追加する方法
### 1. Serviceを追加する
1. `internal/service/` に、`<feature>_service.go` を新規作成。  
    新しいサービスは、以下のようにインターフェース／実装構造体／コンストラクタ関数を最低限用意します。  
    ```go
    package service

    import "keywars/backend/internal/domain/repository"

    // XxxService は、<機能> のユースケースを定義するインターフェース。
    // 将来的にメソッドを追加していく（例：Create, Update, Deleteなど）
    type XxxService interface {
      // 例: CreateXxx(ctx context.Context, input XxxInput) (XxxOutput, error)
    }

    // xxxService は XxxService の具象実装。
    type xxxService struct {
      xxx repository.XxxRepository
    }

    // NewXxxService は、対応する Repository を受け取り、
    // XxxService の実装インスタンスを生成して返す。
    func NewXxxService(xxx repository.XxxRepository) XxxService {
      return &xxxService{
        xxx: xxx,
      }
    }
    ```
2. `internal/domain/repository/` に必要なリポジトリインターフェースを追加（存在しない場合）。  
    新しいリポジトリは、以下のようにインターフェースを最低限用意します。
    ```go
    package repository

    // XxxRepository は、<機能> に関するデータ取得・保存を行うリポジトリのインターフェース。
    // あとでメソッドを追加。
    type XxxRepository interface {}
    ```
3. もし SQL を使う場合は、`internal/infra/repository/sql/` に以下を追加します。
    - `repository.go`：SQL系リポジトリの集約と初期化を行う。
      ```go
      // 
      type Repos struct {
        User repository.UserRepository
        Match repository.MatchRepository // ← 追加
      }

      func New(db *gorm.DB) *Repos {
        return &Repos{
          User: NewUserRepo(db),
          Match: NewMatchRepo(db), // ← 追加
        }
      }
      ```
    - `<feature>_repository.go`：各機能ごとの具体的なリポジトリ実装を記述する。
      ```go
      package sql

      import (
        "gorm.io/gorm"
        "keywars/backend/internal/domain/repository"
      )

      // xxxRepo は、domain 層の XxxRepository を GORM を用いて実装した構造体の定義。
      // データベース操作を担当し、domain 層からの要求を SQL に変換して処理。
      type xxxRepo struct {
        db *gorm.DB
      }

      // NewXxxRepo は、*gorm.DB を受け取り xxxRepo を生成。
      // domain/repository.XxxRepository インターフェースを実装した具体型を返却。
      func NewXxxRepo(db *gorm.DB) repository.XxxRepository {
        return &xxxRepo{db: db}
      }
      ```
3. `internal/app/app.go` の `Repos` にリポジトリを追加。
    ```go
    repos := sqlrepository.Repos {
      User: sqlrepository.NewUserRepo(gormDB),
      Match:  sqlrepository.NewMatchRepository(gormDB), // ← 追加
    }
    ```
4. `internal/app/app.go` の `Services` に新しいサービスを追加。
    ```go
    services := service.Services{
      Auth: service.NewAuthService(repos.User),
      Match: service.NewMatchService(repos.Match), // ← 追加
    }
    ```
5. `internal/service/service.go`にも、構造体を追記。
    ```go
    type Repositories struct {
      User repository.UserRepository
      Match repository.MatchRepository // ← 追加
    }

    type Services struct {
      Auth AuthService
      Match MatchService // ← 追加
    }

    func NewServices(repos Repositories) Services {
      return Services{
        Auth: NewAuthService(repos.User),
        Match: NewMatchService(repos.Match), // ← 追加
      }
    }
    ```

### 2. Handlerを追加する
1. `internal/transport/http/handler/` に `<feature>_handler.go`(feature部分は複数形) を新規作成。  
    新しいハンドラーは、以下のように実装構造体／コンストラクタ関数を最低限用意します。  
    ```go
    package handler

    import "keywars/backend/internal/service"

    // XxxsHandler は、<機能> に関する HTTP リクエストを処理するハンドラ。
    type XxxsHandler struct {
      xxxService service.XxxService
    }

    // NewXxxsHandler は XxxService を受け取り、XxxsHandler を生成。
    func NewXxxsHandler(service service.XxxService) *XxxsHandler {
      return &XxxsHandler{xxxService: service}
    }
    ```
2. `api_handler.go` に依存注入を追加。
    ```go
    type API struct {
      Auth *AuthHandler
      Match *MatchesHandler // ← 追加（複数形）
    }

    func New(services service.Services) *API {
      return &API{
        Auth: NewAuthHandler(services.Auth),
        Match: NewMatchesHandler(services.Match), // ← 追加
      }
    }
    ```
3. `router/router.go` に認証不要のAPI、認証必須のAPIグループどちらかのルートを登録。
    ```go
    // ──────────────────────────────
    // 認証不要のAPI
    // ──────────────────────────────
    e.GET("/matches", api.Matches.GetMatches) // ← どちらかに追加

    // ──────────────────────────────
    // 認証必須のAPIグループ (/api/v1)
    // ──────────────────────────────
    v1 := e.Group("/api/v1", authMiddleware)
    v1.GET("/matches", api.Matches.GetMatches) // ← どちらかに追加
    ```

<br>  

## ハンドラー層でのログ出力とレスポンス方針
### 原則
ハンドラー層では、エラー発生時に **ログ出力とレスポンス返却（h.Respond）をセットで行います**。

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
→ h.Respond のみでOK（ログノイズ防止）

### コーディング例
```go
  logger.Warn().Err(err).Msg("signin failed: invalid credentials")
  return h.Respond(c, http.StatusUnauthorized, echo.Map{
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
│  ├─ service/     # ビジネスロジック層（アプリの振る舞い・ユースケースを記述）
│  ├─ domain/      # データ構造と契約層（Repositoryインターフェース）
│  │  ├─ model/      # ドメインモデル定義（業務ルール中心）
│  │  ├─ port/       # アプリ内外の接続インターフェース（port定義）
│  │  └─ repository/ # Repositoryインターフェース（契約のみを定義）
│  ├─ infra/       # データアクセス層（DBやRedisなど外部リソースへの実装）
│  │  ├─ auth/         # JWTなどの認証関連の実装
│  │  ├─ db/           # データベース接続の初期化や管理を担当
│  │  │  ├─ initial/   # DBマイグレーション後の初期データの投入処理を担当
│  │  │  └─ seed/      # デモ、テストデータ登録関連
│  │  ├─ redis/        # Redis接続の初期化や共通処理を担当
│  │  └─ repository/   # domainで定義したRepositoryの実装層
│  │     ├─ redis/     # Redisを用いたRepositoryの実装
│  │     └─ sql/       # SQL(GORM)を用いたRepositoryの実装
│  │        └─ model/  # DBテーブル構造に対応するGORMモデル定義
│  ├─ transport/   # 通信層（HTTPやWebSocketでリクエストを受ける部分）
│  │  ├─ http/         # HTTP通信関連の処理をまとめる
│  │  │  ├─ handler/       # 各エンドポイントのハンドラを定義
│  │  │  ├─ middleware/    # 認証・ログなどのHTTPミドルウェアを定義
│  │  │  └─ router/        # ルーティング設定を定義
│  │  └─ websocket/    # WebSocket通信関連の処理をまとめる
│  └─ util/         # 汎用的な共通処理をまとめる（アプリ全体から再利用される）
│     ├─ password/       # パスワードハッシュ化・検証などの共通ロジックを提供
│     ├─ validator/      # 入力値の検証（バリデーション）ロジックを提供
│     └─ cookie/         # Cookie操作（設定・削除）を提供
└─ migrations/     # DBマイグレーションSQL（テーブル作成や変更）
```
