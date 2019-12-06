helm "consul" {
  cluster = "cluster.k3s"
  chart = "./helm/consul-helm-0.9.0"
  values = "./helm/consul-values.yaml"
}