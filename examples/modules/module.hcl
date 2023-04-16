module "consul" {
  source = "../container"
}

module "sub_module" {
  source = "./sub_module"
}

module "k8s" {
  depends_on = ["resource.module.consul"]
  source     = "../single_k3s_cluster"
}

module "docs" {
  source = "../docs"
}

module "k8s_exec" {
  disabled   = true
  depends_on = ["container_ingress.consul-container-http-2"]

  source = "../local_exec"
}
