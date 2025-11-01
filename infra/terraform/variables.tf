variable "project_id" {
  type        = string
  description = "The Google Cloud project ID"
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


