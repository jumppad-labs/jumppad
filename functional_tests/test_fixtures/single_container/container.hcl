container "consul" {
  image   {
    name = "consul:1.6.1"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config"
    destination = "/config"
  }

  network    = "network.onprem"
  ip_address = "10.5.0.200" // optional

  resources {
    # Max CPU to consume, 1024 is one core, default unlimited
    cpu = 2048
    # Pin container to specified CPU cores, default all cores
    cpu_pin = [1,2]
    # max memory in MB to consume, default unlimited
    memory = 1024
  }
}