terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "6.8.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

data "google_client_openid_userinfo" "current" {}

output "current_account_email" {
  value       = data.google_client_openid_userinfo.current.email
  description = "Terraform実行に使用されたサービスアカウントまたはユーザーのメールアドレス"
}