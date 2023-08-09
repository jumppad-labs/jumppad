module "consul" {
  source = "../container"
}

module "sub_module" {
  source = "./sub_module"

  variables = {
    version = "1.15.4"
  }
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
  depends_on = ["module.k8s"]

  source = "../local_exec"
}
