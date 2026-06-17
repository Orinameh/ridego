locals {
  db_to_svc = {
    users_db    = "user-service"
    trips_db    = "trip-service"
    payments_db = "payment-service"
  }
}

resource "kubernetes_secret" "db" {
  for_each = var.rds_endpoints

  metadata {
    name      = "${local.db_to_svc[each.key]}-secrets"
    namespace = kubernetes_namespace.ridego.metadata[0].name
  }

  data = {
    DATABASE_URL = replace(each.value, "REDACTED", var.rds_master_password)
  }

  type = "Opaque"
}

resource "kubernetes_secret" "redis" {
  metadata {
    name      = "redis-secrets"
    namespace = kubernetes_namespace.ridego.metadata[0].name
  }

  data = {
    REDIS_URL = "rediss://${var.redis_endpoint}:6379"
  }
}

resource "kubernetes_config_map" "shared" {
  metadata {
    name      = "ridego-config"
    namespace = kubernetes_namespace.ridego.metadata[0].name
  }

  data = {
    NATS_URL    = "nats://nats.ridego.svc.cluster.local:4222"
    CONSUL_ADDR = "consul.ridego.svc.cluster.local:8500"
    LOG_LEVEL   = "info"
    ENVIRONMENT = var.env
  }
}
