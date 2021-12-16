k8s_cluster "dc1" {
  driver  = "k3s" // default

  nodes = 1 // default

  network {
    name = "network.cloud"
  }
}

k8s_cluster "dc2" {
  driver  = "k3s" // default
  version = "v1.18.16"

  nodes = 1 // default

  network {
    name = "network.cloud"
  }
}
