provider "kubernetes" {
  config_path = "~/.kube/config"
}

provider "helm" {
  kubernetes {
    config_path = "~/.kube/config"
  }
}

module "nats" {
  source = "../../modules/nats"
}

module "mongodb" {
  source = "../../modules/mongo"
}
