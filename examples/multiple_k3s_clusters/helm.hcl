variable "consul_helm_version" {
  default = "v0.34.1"
}

helm "consul-dc1" {
  cluster = "k8s_cluster.dc1"
  chart   = "github.com/hashicorp/consul-k8s?ref=${var.consul_helm_version}//charts/consul"
  values  = "./helm/consul-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["release=consul-dc1"]
  }
}

helm "consul-dc2" {
  depends_on = ["helm.consul-dc1"] # run sequentially for slow ci
  cluster = "k8s_cluster.dc2"
  chart   = "github.com/hashicorp/consul-k8s?ref=${var.consul_helm_version}//charts/consul"
  values  = "./helm/consul-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["release=consul-dc2"]
  }
}