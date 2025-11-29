variable "project_id" {
  type        = string
  description = "GooGle Cloud Project ID"
}

variable "region" {
  type        = string
  description = "Region for Cloud Run service"
}

variable "service_name" {
  type        = string
  description = "Cloud Run service name"
}

variable "image" {
  type        = string
  description = "Container image URL"
}

variable "deletion_protection" {
  type        = bool
  description = "Protect from delete"
  default     = false
}

variable "env_vars" {
  type = list(object({
    name  = string
    value = string
  }))
  description = "MySQL environment variables from tfvars"
}

variable "redis_env_vars" {
  type = list(object({
    name  = string
    value = string
  }))
  description = "Redis environment variables from tfvars"
}

variable "secret_env_vars" {
  type = list(object({
    name         = string
    secret_id       = string
    version = string
  }))
  description = "Redis environment variables from secret"
}

variable "volume_mounts" {
  type = list(object({
    name  = string
    mount_path = string
  }))
  description = "volume mounts"
}

variable "vpc_network_name" {
  type        = string
  description = "VPC network name"
}

variable "vpc_subnetwork_name" {
  type        = string
  description = "VPC subnetwork name"
}

variable "network_tags" {
  type        = list(string)
  description = "Network tags"
}

variable "cloudsql_volume_name" {
  type        = string
  description = "Cloud SQL volume name"
}

variable "cloudsql_connection_name" {
  type        = list(string)
  description = "Cloud SQL instance connection name"
}

variable "depends_on_services" {
  type        = list(string)
  description = "List of API services to depend on"
  default     = []
}

