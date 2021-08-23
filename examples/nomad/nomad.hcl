nomad_cluster "dev" {
  client_nodes = "${var.client_nodes}"

  network {
    name = "network.cloud"
  }

  image {
    name = "consul:1.8.0"
  }

  consul_config = "./consul_config/agent.hcl"

  volume {
    source      = "/tmp"
    destination = "/files"
  }
}

nomad_job "consul" {
  cluster = "nomad_cluster.dev"

  paths = ["./app_config/example2.nomad"]
  health_check {
    timeout    = "60s"
    nomad_jobs = ["example_2"]
  }
}
