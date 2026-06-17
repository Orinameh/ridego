variable "project" {
  description = "Project name prefix for all resource tags and names"
  type        = string
  default     = "ridego"
}

variable "env" {
  description = "Deployment environment"
  type        = string
  validation {
    condition     = contains(["staging", "prod"], var.env)
    error_message = "env must be staging or prod"
  }
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-1"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "az_count" {
  description = "Number of availability zones to use"
  type        = number
  default     = 3
}

variable "eks_version" {
  description = "Kubernetes version for EKS"
  type        = string
  default     = "1.30"
}

variable "eks_node_groups" {
  description = "Map of node group name to configuration"
  type = map(object({
    instance_types = list(string)
    desired        = number
    min            = number
    max            = number
    disk_size_gb   = number
  }))
  default = {
    default = {
      instance_types = ["t3.medium"]
      desired        = 3
      min            = 2
      max            = 10
      disk_size_gb   = 50
    }
  }
}

variable "rds_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.t3.medium"
}

variable "rds_databases" {
  description = "List of database names to create on the shared RDS instance"
  type        = list(string)
  default     = ["users_db", "trips_db", "payments_db"]
}

variable "rds_master_password" {
  description = "RDS master user password"
  type        = string
  sensitive   = true
}

variable "redis_node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.small"
}

variable "redis_num_shards" {
  description = "Number of Redis shards. 1 = single node, 2+ = cluster mode"
  type        = number
  default     = 1
}
