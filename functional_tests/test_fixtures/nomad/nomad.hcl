nomad_cluster "dev" {
  version = "v0.10.2"

  nodes = 1 // default

  network {
    name = "network.cloud"
  }

  image {
    name = "consul:1.7.1"
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

nomad_job "redis" {
  cluster = "nomad_cluster.dev"

  paths = ["./app_config/example2.nomad"]
}