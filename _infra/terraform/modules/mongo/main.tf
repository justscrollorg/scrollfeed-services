resource "kubernetes_namespace" "mongo" {
  metadata {
    name = "mongo"
  }
}

resource "kubernetes_persistent_volume_claim" "mongodb_data" {
  depends_on = [kubernetes_namespace.mongo] 

  metadata {
    name      = "mongodb-pvc"
    namespace = kubernetes_namespace.mongo.metadata[0].name
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "10Gi"
      }
    }

    storage_class_name = "linode-block-storage"
  }
}

resource "helm_release" "mongodb" {
  depends_on = [kubernetes_persistent_volume_claim.mongodb_data]

  name             = "mongodb"
  namespace        = kubernetes_namespace.mongo.metadata[0].name
  create_namespace = false 

  repository   = "https://charts.bitnami.com/bitnami"
  chart        = "mongodb"
  version      = "13.15.1"
  reset_values = true

  values = [
    file("${path.module}/values.yaml")
  ]
}
