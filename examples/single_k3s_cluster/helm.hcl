helm "consul" {
  cluster = "k8s_cluster.k3s"
  chart   = "github.com/hashicorp/consul-k8s?ref=v0.34.1//charts/consul"
  values  = "./helm/consul-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["release=consul"]
  }
}

helm "vault" {
  cluster = "k8s_cluster.k3s"
  chart   = "github.com/hashicorp/vault-helm?ref=v0.18.0"

  values  = "./helm/vault-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["app.kubernetes.io/name=vault"]
  }
}
