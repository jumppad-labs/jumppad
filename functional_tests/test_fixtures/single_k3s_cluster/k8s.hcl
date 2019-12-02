cluster "k3s" {
  driver  = "k3s" // default
  version = "v1.0.0"

  nodes = 1 // default

  network = "network.cloud"
}