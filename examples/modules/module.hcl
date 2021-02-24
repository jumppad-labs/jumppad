module "k8s" {
  depends_on = ["module.consul"]
  source = "github.com/shipyard-run/shipyard//examples/single_k3s_cluster?ref=testing"
}

module "consul" {
	source = "../container"
}

container_ingress "consul-container-http-2" {
  target  = "container.consul"

  network {
    name = "network.onprem"
  }

  port {
    local  = 8500
    remote = 8500
    host   = 18600
  }
}

module "docs" {
  depends_on = ["container_ingress.consul-container-http-2"]
	
  source = "../docs"
}
