variable "version" {
  default = "consul:1.6.1"
}

variable "port_range" {
  default = "8500-8502"
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}

resource "template" "consul_config" {

  source = <<-EOF
    data_dir = "{{ data_dir }}"
    log_level = "DEBUG"

    datacenter = "dc1"
    primary_datacenter = "dc1"

    server = true

    bootstrap_expect = 1
    ui = true
  EOF

  destination = "${data("single")}/consul.hcl"

  variables = {
    data_dir = "/tmp"
  }
}

resource "container" "consul" {
  image {
    name = variable.version
  }

  command = ["consul", "agent", "-dev", "-client", "0.0.0.0"]

  network {
    id         = resource.network.onprem.meta.id
    ip_address = "10.6.0.200"
  }

  port_range {
    range       = variable.port_range
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
    source      = resource.template.consul_config.destination
    destination = "/config/config.hcl"
  }

  volume {
    source      = "images.volume.jumppad.dev"
    destination = "/cache"
    type        = "volume"
  }
}

output "consul_addr" {
  value = resource.container.consul.network[0].assigned_address
}