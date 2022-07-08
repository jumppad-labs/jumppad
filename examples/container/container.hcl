variable "consul_version" {
  default = "1.10.6"
}

variable "envoy_version" {
  default = "1.18.4"
}

template "consul_config" {

  source = <<EOF
data_dir = "#{{ .Vars.data_dir }}"
log_level = "DEBUG"

datacenter = "dc1"
primary_datacenter = "dc1"

server = true

bootstrap_expect = 1
ui = true

bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"
advertise_addr = "10.6.0.200"

ports {
  grpc = 8502
}

connect {
  enabled = true
}
EOF

  destination = "./consul_config/consul.hcl"

  vars = {
    data_dir = "/tmp"
  }
}

container "consul_disabled" {
  disabled = true

  image {
    name = "consul:${var.consul_version}"
  }
}

container "consul" {
  image {
    name = "consul:${var.consul_version}"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config/consul.hcl"
    destination = "/config"
  }

  network {
    name       = "network.onprem"
    ip_address = "10.6.0.200" // optional
    aliases    = ["myalias"]
  }

  env {
    key   = "something"
    value = var.something
  }

  env {
    key   = "foo"
    value = env("BAH")
  }

  env {
    key   = "file"
    value = file("./conf.txt")
  }

  resources {
    # Max CPU to consume, 1000 is one core, default unlimited
    cpu = 2000
    # Pin container to specified CPU cores, default all cores
    cpu_pin = [0, 1]
    # max memory in MB to consume, default unlimited
    memory = 1024
  }

  port_range {
    range       = "8500-8502"
    enable_host = true
  }

  env {
    key   = "abc"
    value = "123"
  }

  env {
    key   = "SHIPYARD_FOLDER"
    value = shipyard()
  }

  env {
    key   = "HOME_FOLDER"
    value = home()
  }
}

sidecar "envoy" {
  target = "container.consul"

  image {
    name = "envoyproxy/envoy:v${var.envoy_version}"
  }

  command = ["tail", "-f", "/dev/null"]

  volume {
    source      = "./consul_config"
    destination = "/config"
  }
}
