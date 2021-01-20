local_ingress "tester" {
  target = "k8s_cluster.k3s"
  destination = "localhost"

  port {
    remote = 10000
    local = 30002
  }
}

k8s_ingress "connector-http" {
  cluster = "k8s_cluster.k3s"
  service  = "tester"
  namespace = "shipyard"

  network {
    name = "network.cloud"
  }

  port {
    local  = 10000
    remote = 10000
    host   = 10000
  }
}
