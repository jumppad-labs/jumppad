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

  datacenter = variable.datacenter

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

resource "template" "example_2" {
  source = file("./app_config/example2.nomad")

  variables = {
    datacenter = variable.datacenter
  }

  destination = "${data("jobs")}/example2.nomad"
}

resource "nomad_job" "example_2" {
  cluster = resource.nomad_cluster.dev

  paths = [resource.template.example_2.destination]

  health_check {
    timeout = "60s"
    jobs    = ["example_2"]
  }
}