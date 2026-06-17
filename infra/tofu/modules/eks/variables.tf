variable "project" {
  description = "Project name prefix"
  type        = string
}

variable "env" {
  description = "Deployment environment"
  type        = string
}

variable "cluster_version" {
  description = "Kubernetes version"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnets" {
  description = "Private subnet IDs for worker nodes"
  type        = list(string)
}

variable "node_groups" {
  description = "Map of node group name to configuration"
  type = map(object({
    instance_types = list(string)
    desired        = number
    min            = number
    max            = number
    disk_size_gb   = number
  }))
}

variable "public_access_cidrs" {
  description = "CIDRs allowed to reach the public API endpoint"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}
