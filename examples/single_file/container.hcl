variable "version" {
  default = "consul:1.6.1"
}

network "onprem" {
  subnet = "10.6.0.0/16"
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

EOF

  destination = "${data("single")}/consul.hcl"

  vars = {
    data_dir = "/tmp"
  }
}

container "consul" {
  image {
    name = var.version
  }

  command = ["consul", "agent", "-dev", "-client", "0.0.0.0"]

  network {
    name       = "network.onprem"
    ip_address = "10.6.0.200"
  }

  port_range {
    range       = "8500-8502"
    enable_host = true
  }

  resources {
    # Max CPU to consume, 1024 is one core, default unlimited
    cpu = 2048
    # Pin container to specified CPU cores, default all cores
    cpu_pin = [1]
    # max memory in MB to consume, default unlimited
    memory = 1024
  }

  volume {
    source      = data("temp")
    destination = "/test"
  }

  volume {
    source      = resources.template.consul_config.destination
    destination = "/config/config.hcl"
  }

  volume {
    source      = "images.volume.shipyard.run"
    destination = "/cache"
    type        = "volume"
  }
}


source = resources.template.consul_config.destination
