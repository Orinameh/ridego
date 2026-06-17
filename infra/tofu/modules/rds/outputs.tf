output "endpoint" {
  description = "RDS connection endpoint (host:port)"
  value       = "${aws_db_instance.this.address}:${aws_db_instance.this.port}"
  sensitive   = true
}

output "endpoints" {
  description = "Map of database name to connection string"
  sensitive   = true
  value = {
    for db in var.databases :
    db => "postgres://ridego_master:REDACTED@${aws_db_instance.this.endpoint}/${db}?sslmode=require"
  }
}

output "instance_id" {
  description = "RDS instance identifier"
  value       = aws_db_instance.this.id
}

output "address" {
  description = "RDS instance hostname"
  value       = aws_db_instance.this.address
  sensitive   = true
}

output "port" {
  description = "RDS instance port"
  value       = aws_db_instance.this.port
}
