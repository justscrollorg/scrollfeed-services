resource "kubernetes_namespace" "nats_system" {
  metadata {
    name = "nats-system"
  }
}

resource "kubernetes_persistent_volume_claim" "nats_jetstream" {
  depends_on = [kubernetes_namespace.nats_system]

  metadata {
    name      = "nats-jetstream-pvc"
    namespace = kubernetes_namespace.nats_system.metadata[0].name
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "5Gi"
      }
    }

    storage_class_name = "linode-block-storage"
  }
}

resource "helm_release" "nats" {
  depends_on = [kubernetes_persistent_volume_claim.nats_jetstream]

  name             = "nats"
  namespace        = kubernetes_namespace.nats_system.metadata[0].name
  create_namespace = false # Namespace already created above

  repository = "https://nats-io.github.io/k8s/helm/charts/"
  chart      = "nats"
  version    = "0.19.12"

  values = [
    file("${path.module}/values.yaml")
  ]
}
