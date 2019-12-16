container "consul_nomad" {
  image   {
    name = "consul:1.6.1"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config"
    destination = "/config"
  }

  network    = "network.nomad"
  ip_address = "10.6.0.2" // optional
}

cluster "nomad" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 1 // default

  network = "network.nomad"
  
  config {
    key = "CONSUL_HTTP_ADDR"
    value = "container.consul_nomad.ip_address"
  }
}

ingress "consul_nomad" {
  target = "container.consul_nomad"

  port {
    local  = 8500
    remote = 8500
    host   = 38500
  }

  port {
    local  = 8600
    remote = 8600
  }

  port {
    local  = 8302
    remote = 8302
  }

  port {
    local  = 8301
    remote = 8301
  }

  port {
    local  = 8300
    remote = 8300
  }

  ip_address = "192.168.1.103" // if blank will auto assign, all clusters and containers are on wan
}

ingress "nomad" {
  target  = "cluster.nomad"
  service = "nomad-server" //kubernetes service or nomad job

  port {
    local  = 4646
    remote = 4646
    host   = 14646
  }
}
