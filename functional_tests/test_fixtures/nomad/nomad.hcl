cluster "nomad" {
  driver  = "nomad" // default
  version = "v0.10.2"

  nodes = 1 // default

  network = "network.cloud"

  image {
    name = "consul:1.6.1"
  }

  env {
    key = "CONSUL_SERVER"
    value = "consul.cloud.shipyard"
  }
  
  env {
    key = "CONSUL_DATACENTER"
    value = "dc1"
  }
}