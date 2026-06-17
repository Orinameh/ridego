project  = "ridego"
env      = "prod"
region   = "eu-west-1"
vpc_cidr = "10.0.0.0/16"
az_count = 3

eks_version = "1.30"
eks_node_groups = {
  default = {
    instance_types = ["t3.large"]
    desired        = 3
    min            = 3
    max            = 12
    disk_size_gb   = 80
  }
  spot = {
    instance_types = ["t3.large", "t3a.large"]
    desired        = 2
    min            = 0
    max            = 8
    disk_size_gb   = 60
  }
}

rds_instance_class = "db.t3.medium"
rds_databases      = ["users_db", "trips_db", "payments_db"]

redis_node_type  = "cache.t3.medium"
redis_num_shards = 2
