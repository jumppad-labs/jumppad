helm "consul" {
  cluster = "k8s_cluster.k3s"
  chart = "./helm/consul-helm-0.9.0"
  values = "./helm/consul-values.yaml"
  
  health_check {
    timeout = "60s"
    pods = ["release=consul"]
  }
}