variable "consul_version" {
  default = "1.10.6"
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}

resource "container" "consul" {
  image {
    name = "consul:${variable.consul_version}"
  }

  command = ["consul", "agent"]

  network {
    id         = resource.network.onprem.id
    ip_address = "10.6.0.200" // optional
    aliases    = ["myalias"]
  }
}

resource "remote_exec" "in_container" {
  target = resource.container.consul.id

  script = <<-EOF
  #/bin/sh -e

  ls -las
  EOF
}

resource "remote_exec" "standalone" {
  image {
    name = "consul:${variable.consul_version}"
  }

  script = <<-EOF
  #/bin/sh -e

  ls -las
  EOF
}