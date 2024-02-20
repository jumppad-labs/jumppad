module "consul" {
  source = "./modules/container"
}

module "sub_module" {
  source = "./modules/sub_module"

  variables = {
    version = "1.15.4"
  }
}

module "k8s" {
  depends_on = ["resource.module.consul"]
  source     = "./modules/single_k3s_cluster"
}

module "docs" {
  source = "./modules/docs"
}

module "k8s_exec" {
  disabled   = true
  depends_on = ["module.k8s"]

  source = "./modules/exec"
}
