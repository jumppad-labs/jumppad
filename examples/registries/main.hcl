resource "network" "cloud" {
  subnet = "10.6.0.0/16"
}

module "nomad" {
  disabled = !variable.nomad_enabled

  depends_on = ["resource.build.app"]
  source     = "./nomad"

  variables = {
    network_id = resource.network.cloud.id
  }
}

module "k8s" {
  disabled = !variable.k8s_enabled

  depends_on = ["resource.build.app"]
  source     = "./k8s"

  variables = {
    network_id = resource.network.cloud.id
  }
}

output "KUBECONFIG" {
  disabled = !variable.k8s_enabled

  value = module.k8s.output.KUBECONFIG
}