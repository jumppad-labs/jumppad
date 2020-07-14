container "consul" {
  image   {
    name = "consul:${env("CONSUL_VERSION")}"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config"
    destination = "/config"
  }

  network {
    name = "network.onprem"
    ip_address = "10.6.0.200" // optional
  }

  env {
    key = "something"
    value = "${something}"
  }

  resources {
    # Max CPU to consume, 1024 is one core, default unlimited
    cpu = 2048
    # Pin container to specified CPU cores, default all cores
    cpu_pin = [1,2]
    # max memory in MB to consume, default unlimited
    memory = 1024
  }

  port_range {
    range       = "8500-8502"
    enable_host = true
  }

  env {
    key ="abc"
    value = "123"
  }
  
  env {
    key ="SHIPYARD_FOLDER"
    value = "${shipyard()}"
  }
  
  env {
    key ="HOME_FOLDER"
    value = "${home()}"
  }
}

sidecar "envoy" {
  target = "container.consul"

  image   {
    name = "envoyproxy/envoy-alpine:v${env("ENVOY_VERSION")}"
  }

  command = ["tail", "-f", "/dev/null"]
  
  volume {
    source      = "./consul_config"
    destination = "/config"
  }
}