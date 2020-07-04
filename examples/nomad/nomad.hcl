nomad_cluster "dev" {
  version = "v0.11.2"

  nodes = 1 // default

  network {
    name = "network.cloud"
  }

  image {
    name = "consul:1.8.0"
  }

  volume {
    source = "/tmp"
    destination = "/files"
  }

  env {
    key = "CONSUL_SERVER"
    value = "consul.container.shipyard.run"
  }
  
  env {
    key = "CONSUL_DATACENTER"
    value = "dc1"
  }
}

nomad_job "consul" {
  cluster = "nomad_cluster.dev"

  paths = ["./app_config/example2.nomad"]
  health_check {
    timeout = "60s"
    nomad_jobs = ["example_2"]
  }
}