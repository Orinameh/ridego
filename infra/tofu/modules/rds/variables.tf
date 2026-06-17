variable "project" {
  description = "Project name prefix"
  type        = string
}

variable "env" {
  description = "Deployment environment"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "database_subnets" {
  description = "Isolated subnet IDs for RDS"
  type        = list(string)
}

variable "db_subnet_group_name" {
  description = "RDS subnet group name"
  type        = string
}

variable "allowed_sg_id" {
  description = "Security group ID allowed to connect to RDS (EKS nodes)"
  type        = string
}

variable "instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.medium"
}

variable "databases" {
  description = "List of database names to create"
  type        = list(string)
}

variable "master_password" {
  description = "RDS master password"
  type        = string
  sensitive   = true
}
