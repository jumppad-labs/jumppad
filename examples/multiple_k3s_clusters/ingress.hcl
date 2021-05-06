ingress "consul-http-dc1" {
  source {
    driver = "local"
    
    config {
      port = 18500
    }
  }
  
  destination {
    driver = "k8s"
    
    config {
      cluster = "k8s_cluster.dc1"
      address = "consul-dc1-consul-server.default.svc"
      port = 8500
    }
  }
}


ingress "consul-http-dc2" {
  source {
    driver = "local"
    
    config {
      port = 18501
    }
  }
  
  destination {
    driver = "k8s"
    
    config {
      cluster = "k8s_cluster.dc2"
      address = "consul-dc2-consul-server.default.svc"
      port = 8500
    }
  }
}