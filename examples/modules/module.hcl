module "consul" {
  source = "../container"
}

module "sub_module" {
  source = "./sub_module"
}

module "k8s" {
  depends_on = ["module.consul"]
  source     = "../single_k3s_cluster"
}

container_ingress "consul-container-http-2" {
  target = "container.consul"

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

module "k8s_exec" {
  disabled   = true
  depends_on = ["container_ingress.consul-container-http-2"]

  source = "../local_exec"
}
