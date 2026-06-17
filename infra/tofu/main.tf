# Root module — composes VPC, EKS, RDS, and ElastiCache.
# Run: tofu init && tofu apply -var-file=environments/prod.tfvars

module "vpc" {
  source = "./modules/vpc"

  project  = var.project
  env      = var.env
  region   = var.region
  cidr     = var.vpc_cidr
  az_count = var.az_count
}

module "eks" {
  source = "./modules/eks"

  project         = var.project
  env             = var.env
  cluster_version = var.eks_version
  vpc_id          = module.vpc.vpc_id
  private_subnets = module.vpc.private_subnet_ids
  node_groups     = var.eks_node_groups

  depends_on = [module.vpc]
}

module "rds" {
  source = "./modules/rds"

  project              = var.project
  env                  = var.env
  vpc_id               = module.vpc.vpc_id
  database_subnets     = module.vpc.database_subnet_ids
  db_subnet_group_name = module.vpc.db_subnet_group_name
  allowed_sg_id        = module.eks.node_security_group_id
  instance_class       = var.rds_instance_class
  databases            = var.rds_databases
  master_password      = var.rds_master_password

  depends_on = [module.vpc]
}

module "elasticache" {
  source = "./modules/elasticache"

  project         = var.project
  env             = var.env
  vpc_id          = module.vpc.vpc_id
  private_subnets = module.vpc.private_subnet_ids
  allowed_sg_id   = module.eks.node_security_group_id
  node_type       = var.redis_node_type
  num_shards      = var.redis_num_shards

  depends_on = [module.vpc]
}

# Writes connection strings into Kubernetes Secrets after cluster is ready
module "k8s_secrets" {
  source = "./modules/k8s-secrets"

  env                 = var.env
  cluster_endpoint    = module.eks.cluster_endpoint
  cluster_ca          = module.eks.cluster_ca
  cluster_token       = module.eks.cluster_token
  rds_endpoints       = module.rds.endpoints
  redis_endpoint      = module.elasticache.primary_endpoint
  rds_master_password = var.rds_master_password

  depends_on = [module.eks, module.rds, module.elasticache]
}
