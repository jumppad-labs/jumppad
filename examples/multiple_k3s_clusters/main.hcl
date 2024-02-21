resource "network" "cloud" {
  subnet = "10.5.0.0/16"
}

module "consul_dc1" {
  source = "./module_k3s"

  variables = {
    network_id  = resource.network.cloud.meta.id
    consul_port = 18500
  }
}

module "consul_dc2" {
  // CI has limited resources, add a manual dependency to ensure that only one module
  // is created at once
  depends_on = ["module.consul_dc1"]

  source = "./module_k3s"

  variables = {
    network_id  = resource.network.cloud.meta.id
    consul_port = 18501
  }
}

output "dc1_id" {
  value = module.consul_dc1.output.k8s_port
}

output "addr_map" {
  value = {
    dc1 = module.consul_dc1.output.consul_http_addr
    dc2 = module.consul_dc2.output.consul_http_addr
  }
}

output "addr_list" {
  value = [
    module.consul_dc1.output.consul_http_addr,
    module.consul_dc2.output.consul_http_addr
  ]
}