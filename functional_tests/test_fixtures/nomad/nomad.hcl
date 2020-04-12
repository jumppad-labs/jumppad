nomad_cluster "dev" {
  version = "v0.10.2"

  nodes = 1 // default

  network {
    name = "network.cloud"
  }

  image {
    name = "consul:1.7.1"
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

nomad_job "redis" {
  cluster = "nomad_cluster.dev"

  paths = ["./app_config/example2.nomad"]
  health_check {
    timeout = "60s"
    nomad_jobs = ["example_2"]
  }
}