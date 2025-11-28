variable "project_name" {
  type        = string
  description = "The Google Cloud project name in lowercase"
}

variable "project_id" {
  type        = string
  description = "The Google Cloud project ID"
}

variable "project_number" {
  type        = number
  description = "The Google Cloud project number"
}

variable "region" {
  type        = string
  description = "Region for resources"
}

variable "dns_zone_name" {
  type        = string
  description = "Cloud DNS zone name"
}

variable "dns_record_name" {
  type        = string
  description = "Cloud DNS FQDN"
}

variable "frontend_static_path" {
  type        = string
  description = "Path to the built frontend static files"
}

variable "mysql_user" {
  type        = string
  description = "User for mysql instance"
}

variable "mysql_root_password" {
  type        = string
  description = "Root password for mysql instance"
  sensitive   = true
}

variable "mysql_user_password" {
  type        = string
  description = "Password for mysql user"
  sensitive   = true
}

variable "mysql_database" {
  type        = string
  description = "Database name for mysql instance"
}

variable "mysql_port" {
  type        = number
  description = "Port for mysql instance"
}

variable "redis_password" {
  type        = string
  description = "Password for redis instance"
  sensitive   = true
}

variable "github_token_secret_name" {
  type        = string
  description = "Secret name with Github oauth token"
  sensitive   = true
}

variable "installed_id" {
  type        = number
  description = "Google Cloud Build app on GitHub Apps"
}

variable "repository_name" {
  type        = string
  description = "My github repository name"
}

variable "repository_uri" {
  type        = string
  description = "My github repository uri"
}

variable "ar-repository_name" {
  type        = string
  description = "Repogitory Name for Cloud Run in Artifact Registry"
}