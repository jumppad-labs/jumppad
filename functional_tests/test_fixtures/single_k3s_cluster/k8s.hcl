cluster "k3s" {
  driver  = "k3s" // default
  version = "v1.0.0"

  nodes = 1 // default

  network {
    name = "cloud"
  }

  image {
    name = "consul:1.6.1"
  }
}