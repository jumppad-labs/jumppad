k8s_ingress "consul-http" {
  cluster = "k8s_cluster.k3s"
  service  = "consul-consul-server"

  network {
    name = "network.cloud"
  }

  port {
    local  = 8500
    remote = 8500
    host   = 18500
  }
}

k8s_ingress "vault-http" {
  cluster = "k8s_cluster.k3s"
  service  = "vault"

  network {
    name = "network.cloud"
  }

  port {
    local  = 8200
    remote = 8200
    host   = 18200
  }
}