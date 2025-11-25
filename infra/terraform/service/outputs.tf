output "current_account_email" {
  value       = data.google_client_openid_userinfo.current.email
  description = "Terraform実行に使用されたサービスアカウントまたはユーザーのメールアドレス"
}