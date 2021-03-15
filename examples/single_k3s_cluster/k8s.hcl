k8s_cluster "k3s" {
  driver  = "k3s" // default

  nodes = 1 // default

  network {
    name = "network.cloud"
  }

  image {
    name = "shipyardrun/connector:v0.0.10"
  }
}
