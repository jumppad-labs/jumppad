template "nomad_config" {

  source = <<-EOS
  plugin "docker" {
    config {
      allow_privileged = true
      volumes {
        enabled = true
        selinuxlabel = "z"
      }
    }
  }
  EOS

  destination = "${data("nomad-config")}/user_config.hcl"
}

nomad_cluster "dev" {
  client_nodes = "${var.client_nodes}"

  client_config = "${data("nomad-config")}/user_config.hcl"

  network {
    name = "network.cloud"
  }

  image {
    name = "consul:1.10.1"
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
