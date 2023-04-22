// consul http to localhost:8500 on kubernetes
resource "ingress" "consul-http-kubernetes" {
  port = 18500 

  target {
    id = resource.k8s_cluster.k3s.id
    port = 8500
    
    service = "consul-server"
    namespace = "default"
  }
  
  // available for remote connector
  public = true
}

// consul http to localhost:8500 on nomad
resource "ingress" "consul-http-kubernetes" {
  port = 18501

  target {
    id = resource.nomad_cluster.dev.id
    named_port = "http"
    
    job = "consul"
    group = "consul"
    task = "consul"
  }
  
  // not available for remote connector
  public = false
}

// consul http to localhost:8500 on docker
resource "ingress" "consul-http-docker" {
  port = 18502

  target {
    id = resource.container.consul.id
    port = 8500
  }

  // not available for remote connector
  public = false
}

// consul http on local machine on the network cloud
resource "egress" "local-app" {
  address = "localhost"
  port = 8500

  virtual_service {
    network_id = resource.network.cloud.id
    port = 8500
  }
  
  // available for remote connector
  public = true
}

// exposes public ports from a remote blueprint to the 
// local application
resource "remote" "remote_consul"
  token = "dfdfdfdfd343434cdfdf"
}