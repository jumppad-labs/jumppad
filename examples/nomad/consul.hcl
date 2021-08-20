container "consul" {
  image {
    name = "consul:1.10.1"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config/server.hcl"
    destination = "/config/config.hcl"
  }

  network {
    name = "network.cloud"
  }
}