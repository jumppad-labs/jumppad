cluster "cloud" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 1 // default

  network = "network.k8s"
}

helm "consul" {
  cluster = "cluster.cloud"
  chart   = "${env("SHIPYARD_HOME")}/helm/charts/consul"
  values  = "${env("SHIPYARD_HOME")}/helm/charts/consul-values.yml"

  health_check {
    timeout = "2m"
    http     = "http://consul-consul:8500/v1/leader"                          // can the http endpoint be reached
    tcp      = "consul-consul:8500"                                           // can a TCP connection be made
    services = ["consul-consul"]                                              // does service exist and there are endpoints
    pods     = ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
  }
}

// runs kubectl apply
k8s_config "dashboard" {
  cluster = "cluster.cloud"
  path  = "${env("SHIPYARD_HOME")}/k8s_config/dashboard.yml"
  wait_until_ready = false

  health_check {
    timeout = "2m"
    http     = "http://consul-consul:8500/v1/leader"                          // can the http endpoint be reached
    tcp      = "consul-consul:8500"                                           // can a TCP connection be made
    services = ["consul-consul"]                                              // does service exist and there are endpoints
    pods     = ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
  }
}

ingress "k8s-dashboard" {
  target  = "cluster.cloud"
  service = "kubernetes-dashboard" //kubernetes service or nomad job

  port {
    local  = 8443
    remote = 8443
    host   = 18443
  }
}

ingress "consul-k8s" {
  target  = "cluster.cloud"
  service = "consul-consul-server" //kubernetes service or nomad job

  port {
    local  = 8500
    remote = 8500
    host   = 28500
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

  ip_address = "192.168.1.123" // if blank will auto assign, all clusters and containers are on wan
}

ingress "gateway-k8s" {
  target  = "cluster.cloud"
  service = "consul-consul-mesh-gateway" //kubernetes service or nomad job

  port {
    local  = 443
    remote = 443
  }

  ip_address = "192.169.7.240" // if ommited will auto assign
}
