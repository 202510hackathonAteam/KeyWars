# 本番設定の概要

## Cookie設定について
`backend/internal/config/config.go`
開発環境では `Secure=false`、`Domain=""` のままで動作します。  
本番環境では必ず以下のように設定してください：

| 設定項目 | 本番値の例 | 説明 |
|-----------|-------------|------|
| `CookieConfig.Domain` | `"example.com"` | 配置先ドメイン名 |
| `CookieConfig.SameSite` | `http.SameSiteStrictMode` | 同一サイト内のみで Cookie を送信する設定 |
| `CookieConfig.Secure` | `true` | HTTPS通信専用Cookieにする（セキュリティ必須） |

> 注意: `Secure=false` のままの場合、HTTP通信でもトークンが送信されるため危険です。

<br>

## 開発用テストデータの削除
`cmd/seed/main.go`
開発時にはデモユーザーやテスト用データを投入するため、以下のようなコードが存在します。
この処理は 本番環境では不要かつ危険 なので、必ず削除またはコメントアウトしてください。
```go
if err := seed.SeedDevelopmentData(gormDB); err != nil {
  log.Fatalf("failed to load demo data: %v", err)
}
```