variable "env" {
  description = "Deployment environment"
  type        = string
}

variable "cluster_endpoint" {
  description = "EKS cluster API endpoint"
  type        = string
}

variable "cluster_ca" {
  description = "EKS cluster certificate authority data"
  type        = string
}

variable "cluster_token" {
  description = "EKS cluster auth token"
  type        = string
  sensitive   = true
}

variable "rds_endpoints" {
  description = "Map of database name to connection string"
  type        = map(string)
  sensitive   = true
}

variable "redis_endpoint" {
  description = "Redis primary endpoint"
  type        = string
  sensitive   = true
}

variable "rds_master_password" {
  description = "RDS master password for interpolating into connection strings"
  type        = string
  sensitive   = true
}
