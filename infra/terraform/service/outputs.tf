output "current_account_email" {
  value       = data.google_client_openid_userinfo.current.email
  description = "email address of the service account or user used to run Terraform"
}