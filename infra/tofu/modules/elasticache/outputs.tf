output "primary_endpoint" {
  description = "Redis primary endpoint (cluster mode: configuration endpoint)"
  sensitive   = true
  value = (
    var.num_shards > 1
    ? aws_elasticache_replication_group.this.configuration_endpoint_address
    : aws_elasticache_replication_group.this.primary_endpoint_address
  )
}

output "port" {
  description = "Redis port"
  value       = 6379
}

output "replication_group_id" {
  description = "ElastiCache replication group ID"
  value       = aws_elasticache_replication_group.this.id
}
