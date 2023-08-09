variable "consul_version" {
  default = "1.10.6"
}

variable "envoy_version" {
  default = "1.18.4"
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

  destination = "${data("config")}/consul.hcl"

  variables = {
    data_dir = "/tmp"
  }
}

resource "container" "consul_disabled" {
  disabled = true

  image {
    name = "consul:${variable.consul_version}"
  }
}

resource "container" "consul" {
  image {
    name = "consul:${variable.consul_version}"
  }

  command = ["consul", "agent", "-config-file", "/config/config.hcl"]

  volume {
    source      = "./"
    destination = "/files"
  }

  volume {
    source      = resource.template.consul_config.destination
    destination = "/config/config.hcl"
  }

  network {
    id         = resource.network.onprem.id
    ip_address = "10.6.0.200" // optional
    aliases    = ["myalias"]
  }

  environment = {
    something       = variable.something
    foo             = env("BAH")
    file            = file("./conf.txt")
    abc             = "123"
    SHIPYARD_FOLDER = jumppad()
    HOME_FOLDER     = home()
  }

  resources {
    # Max CPU to consume, 1000 is one core, default unlimited
    cpu = 200
    # Pin container to specified CPU cores, default all cores
    cpu_pin = [0, 1]
    # max memory in MB to consume, default unlimited
    memory = 1024
  }

  port_range {
    range       = "8500-8502"
    enable_host = true
  }

  health_check {
    timeout = "30s"

    http {
      address       = "http://localhost:8500"
      success_codes = [200]
    }

    tcp {
      address = "localhost:8500"
    }

    exec {
      script = <<-EOF
        #!/bin/sh -e

        ls -las
      EOF
    }
  }

}

resource "sidecar" "envoy" {
  target = resource.container.consul

  image {
    name = "envoyproxy/envoy:v${variable.envoy_version}"
  }

  command = ["tail", "-f", "/dev/null"]

  volume {
    source      = data("config")
    destination = "/config"
  }
}