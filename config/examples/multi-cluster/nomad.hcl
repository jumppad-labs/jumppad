container "consul" {
  image   = "consul:1.6.1"
  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config"
    destination = "/config"
  }

  network    = network.nomad
  ip_address = "10.6.0.2" // optional
}

cluster "nomad" {
  driver = "nomad"
  image  = "0.10.0"

  nodes = 3

  network = network.nomad

  config {
    consul_http_addr = container.consul.ip_address
  }
}

ingress "consul" {
  cluster = container.consul
  service = "consul" //kubernetes service or nomad job

  ports {
    local  = 8500
    remote = 8500
    host   = 38500
  }

  ports {
    local  = 8600
    remote = 8600
  }

  ports {
    local  = 8302
    remote = 8302
  }

  ports {
    local  = 8301
    remote = 8301
  }

  ports {
    local  = 8300
    remote = 8300
  }

  ip_address = "192.168.1.103" // if blank will auto assign, all clusters and containers are on wan
}

ingress "nomad" {
  cluster = cluster.nomad
  service = "nomad-server" //kubernetes service or nomad job

  ports {
    local  = 4646
    remote = 4646
    host   = 14646
  }
}
