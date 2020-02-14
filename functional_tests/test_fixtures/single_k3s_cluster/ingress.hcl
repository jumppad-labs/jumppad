ingress "consul-http" {
  target = "k8s_cluster.k3s"
  service  = "svc/consul-consul-server"

  network {
    name = "network.cloud"
  }

  port {
    local  = 8500
    remote = 8500
    host   = 18500
  }
}