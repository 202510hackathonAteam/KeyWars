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

variable "mysql_env_vars" {
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

variable "ingress" {
  type        = string
  description = "Ingress settings"
  default     = "INGRESS_TRAFFIC_ALL" # IAMチェックを無効化
}

variable "depends_on_services" {
  type        = list(string)
  description = "List of API services to depend on"
  default     = []
}

# WebSocket用の追加設定用(仮)
variable "max_instance_count" {
  type        = number
  description = "Maximum number of container instances"
  default     = null
}

variable "min_instance_count" {
  type        = number
  description = "Minimum number of container instances"
  default     = null
}

variable "cpu_limit" {
  type        = string
  description = "CPU limit"
  default     = null
}

variable "memory_limit" {
  type        = string
  description = "Memory limit"
  default     = null
}

variable "timeout_seconds" {
  type        = number
  description = "Request timeout in seconds"
  default     = null
}

variable "execution_environment" {
  type        = string
  description = "Execution environment (gen1 or gen2)"
  default     = null
}