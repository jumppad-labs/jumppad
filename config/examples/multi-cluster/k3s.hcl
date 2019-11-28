cluster "cloud" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 1 // default

  network = network.k8s
}

helm "consul" {
  cluster = cluster.cloud
  chart   = "${environment(SHIPYARD_HOME)}/helm/charts/consul"
  values  = "${environment(SHIPYARD_HOME)}/helm/charts/consul-values.yml"

  health_check {
    http     = "http://consul-consul:8500/v1/leader"                          // can the http endpoint be reached
    tcp      = "consul-consul:8500"                                           // can a TCP connection be made
    services = ["consul-consul"]                                              // does service exist and there are endpoints
    pods     = ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
  }
}

// runs kubectl apply
k8s_config "dashboard" {
  cluster = cluster.cloud
  config  = "${environment(SHIPYARD_HOME)}/k8s_config/dashboard.yml"

  healthcheck {
    http     = "http://consul-consul:8500/v1/leader"                          // can the http endpoint be reached
    tcp      = "consul-consul:8500"                                           // can a TCP connection be made
    services = ["consul-consul"]                                              // does service exist and there are endpoints
    pods     = ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
  }
}

ingress "k8s-dashboard" {
  cluster = cluster.cloud
  service = "kubernetes-dashboard" //kubernetes service or nomad job

  ports {
    local  = 8443
    remote = 8443
    host   = 18443
  }
}

ingress "consul-k8s" {
  cluster = cluster.cloud
  service = "consul-consul-server" //kubernetes service or nomad job

  ports {
    local  = 8500
    remote = 8500
    host   = 28500
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

  ip_address = "192.168.1.123" // if blank will auto assign, all clusters and containers are on wan
}

ingress "gateway-k8s" {
  cluster = cluster.cloud
  service = "consul-consul-mesh-gateway" //kubernetes service or nomad job

  ports {
    local  = 443
    remote = 443
  }

  ip_address = "192.169.7.240" // if ommited will auto assign
}
