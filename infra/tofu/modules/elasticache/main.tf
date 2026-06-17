resource "aws_security_group" "redis" {
  name        = "${var.project}-${var.env}-redis-sg"
  description = "ElastiCache Redis — allow from EKS nodes only"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Redis from EKS nodes"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [var.allowed_sg_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_elasticache_subnet_group" "this" {
  name       = "${var.project}-${var.env}-redis"
  subnet_ids = var.private_subnets
}

resource "aws_elasticache_parameter_group" "this" {
  name   = "${var.project}-${var.env}-redis7"
  family = "redis7"

  parameter {
    name  = "maxmemory-policy"
    value = "allkeys-lru"
  }
  parameter {
    name  = "notify-keyspace-events"
    value = "Ex"
  }
}

resource "aws_elasticache_replication_group" "this" {
  replication_group_id = "${var.project}-${var.env}"
  description          = "RideGo ${var.env} Redis cluster"

  engine         = "redis"
  engine_version = "7.1"
  node_type      = var.node_type
  port           = 6379

  num_node_groups         = var.num_shards
  replicas_per_node_group = var.env == "prod" ? 1 : 0

  automatic_failover_enabled = var.env == "prod" ? true : false
  multi_az_enabled           = var.env == "prod" ? true : false

  subnet_group_name    = aws_elasticache_subnet_group.this.name
  security_group_ids   = [aws_security_group.redis.id]
  parameter_group_name = aws_elasticache_parameter_group.this.name

  at_rest_encryption_enabled = true
  transit_encryption_enabled = true

  snapshot_retention_limit = var.env == "prod" ? 5 : 1
  snapshot_window          = "02:00-03:00"
  maintenance_window       = "mon:05:00-mon:06:00"

  apply_immediately = var.env != "prod"

  tags = { Name = "${var.project}-${var.env}-redis" }
}

resource "aws_cloudwatch_metric_alarm" "redis_cpu" {
  alarm_name          = "${var.project}-${var.env}-redis-cpu-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "EngineCPUUtilization"
  namespace           = "AWS/ElastiCache"
  period              = 300
  statistic           = "Average"
  threshold           = 70
  dimensions          = { ReplicationGroupId = aws_elasticache_replication_group.this.id }
}
