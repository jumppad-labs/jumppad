resource "template" "nomad_config" {

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

resource "nomad_cluster" "dev" {
  client_nodes = variable.client_nodes

  client_config = resource.template.nomad_config.destination
  consul_config = "./consul_config/agent.hcl"

  network {
    id = resource.network.cloud.id
  }

  copy_image {
    name = "consul:1.10.1"
  }

  volume {
    source      = "/tmp"
    destination = "/files"
  }
}

resource "nomad_job" "example_2" {
  cluster = resource.nomad_cluster.dev.id

  paths = ["./app_config/example2.nomad"]

  health_check {
    timeout = "60s"
    jobs    = ["example_2"]
  }
}