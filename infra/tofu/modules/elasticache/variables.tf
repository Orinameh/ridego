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

variable "private_subnets" {
  description = "Private subnet IDs for ElastiCache"
  type        = list(string)
}

variable "allowed_sg_id" {
  description = "Security group ID allowed to connect to Redis (EKS nodes)"
  type        = string
}

variable "node_type" {
  description = "ElastiCache node type"
  type        = string
  default     = "cache.t3.small"
}

variable "num_shards" {
  description = "Number of shards. 1 = single primary, 2+ = cluster mode"
  type        = number
  default     = 1
}
