k8s_cluster "k3s" {
  driver  = "k3s" // default
  version = "v1.17.4-k3s1"

  nodes = 1 // default

  network {
    name = "network.cloud"
  }

  image {
    name = "consul:1.6.1"
  }
}