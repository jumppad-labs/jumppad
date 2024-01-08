resource "container" "consul" {
  image {
    name = "consul:1.10.6"
  }

  command = ["consul", "agent", "-config-file=/config/config.hcl"]

  volume {
    source      = "./consul_config/server.hcl"
    destination = "/config/config.hcl"
  }

  network {
    id = resource.network.cloud.resource_id
  }

  port {
    local  = 8500
    remote = 8500
    host   = 18500
  }
}