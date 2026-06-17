resource "kubernetes_namespace" "ridego" {
  metadata {
    name = "ridego"
    labels = {
      name        = "ridego"
      environment = var.env
    }
  }
}

resource "kubernetes_resource_quota" "ridego" {
  metadata {
    name      = "ridego-quota"
    namespace = kubernetes_namespace.ridego.metadata[0].name
  }
  spec {
    hard = {
      "pods"            = "100"
      "requests.cpu"    = "20"
      "requests.memory" = "40Gi"
      "limits.cpu"      = "40"
      "limits.memory"   = "80Gi"
    }
  }
}
