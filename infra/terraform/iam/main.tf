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


# terraform用サービスアカウント
resource "google_service_account" "terraform_user" {
  account_id   = "terraform-user"
  display_name = "Terraform Service Account"
  description  = "Service account for terraform"
}

# Cloud DNS 権限
resource "google_project_iam_member" "dns_admin" {
  project = var.project_id
  role    = "roles/dns.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Compute Engine 権限
resource "google_project_iam_member" "compute_admin" {
  project = var.project_id
  role    = "roles/compute.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Cloud Storage 権限
resource "google_project_iam_member" "storage_admin" {
  project = var.project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Cloud Run 権限
resource "google_project_iam_member" "run_admin" {
  project = var.project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Cloud SQL 権限
resource "google_project_iam_member" "cloudsql_admin" {
  project = var.project_id
  role    = "roles/cloudsql.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Memorystore (Redis) 権限
resource "google_project_iam_member" "redis_admin" {
  project = var.project_id
  role    = "roles/redis.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Secret Manager 権限
resource "google_project_iam_member" "secretmanager_admin" {
  project = var.project_id
  role    = "roles/secretmanager.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# VPC 権限
resource "google_project_iam_member" "network_admin" {
  project = var.project_id
  role    = "roles/compute.networkAdmin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Artifact Registry 権限
resource "google_project_iam_member" "artifact_registry_admin" {
  project = var.project_id
  role    = "roles/artifactregistry.admin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}

# Service Usage Admin 権限 (API有効化用)
resource "google_project_iam_member" "service_usage_admin" {
  project = var.project_id
  role    = "roles/serviceusage.serviceUsageAdmin"
  member  = "serviceAccount:${google_service_account.terraform_user.email}"
}