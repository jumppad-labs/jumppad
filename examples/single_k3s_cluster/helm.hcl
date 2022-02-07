helm "consul" {
  cluster = "k8s_cluster.k3s"

  # When no repositroy is specified, either a local path or go getter URL
  chart = "github.com/hashicorp/consul-k8s?ref=v0.34.1//charts/consul"

  values = "./helm/consul-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["release=consul"]
  }
}

helm "vault" {
  depends_on = ["helm.consul"] # only install one at a time

  cluster = "k8s_cluster.k3s"

  repository {
    name = "hashicorp"
    url  = "https://helm.releases.hashicorp.com"
  }

  chart   = "hashicorp/vault" # When repository specified this is the name of the chart
  version = "v0.18.0"         # Version of the chart when repository specified

  values = "./helm/vault-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["app.kubernetes.io/name=vault"]
  }
}
