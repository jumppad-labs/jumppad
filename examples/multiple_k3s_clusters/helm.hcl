helm "consul-dc1" {
  cluster = "k8s_cluster.dc1"
  chart = "./helm/consul-helm-0.22.0"
  values = "./helm/consul-values.yaml"
  
  health_check {
    timeout = "240s"
    pods = ["release=consul-dc1"]
  }
}

helm "consul-dc2" {
  cluster = "k8s_cluster.dc2"
  chart = "./helm/consul-helm-0.22.0"
  values = "./helm/consul-values.yaml"
  
  health_check {
    timeout = "240s"
    pods = ["release=consul-dc2"]
  }
}