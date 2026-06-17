project  = "ridego"
env      = "staging"
region   = "eu-west-1"
vpc_cidr = "10.1.0.0/16"
az_count = 2

eks_version = "1.30"
eks_node_groups = {
  default = {
    instance_types = ["t3.medium"]
    desired        = 2
    min            = 1
    max            = 4
    disk_size_gb   = 40
  }
}

rds_instance_class = "db.t3.small"
rds_databases      = ["users_db", "trips_db", "payments_db"]

redis_node_type  = "cache.t3.micro"
redis_num_shards = 1
