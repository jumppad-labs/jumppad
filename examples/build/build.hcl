variable "container_enabled" {
  default = true
}

variable "nomad_enabled" {
  default = false
}

variable "kubernetes_enabled" {
  default = false
}

// use a random ingress by default
variable "container_ingress_port" {
  default = 0
}

variable "nomad_ingress_port" {
  default = 0
}

variable "kubernetes_ingress_port" {
  default = 0
}

resource "build" "app" {
  container {
    dockerfile = "Dockerfile"
    context    = "./src"
  }

  output {
    source      = "/bin/app"
    destination = "${data("output_file")}/app"
  }

  output {
    source      = "/bin"
    destination = "${data("output_dir")}/bin"
  }
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}

module "container" {
  disabled = !variable.container_enabled
  source   = "./container"

  variables = {
    image        = resource.build.app.image
    network      = resource.network.onprem.id
    ingress_port = variable.container_ingress_port
  }
}

module "nomad" {
  disabled = !variable.nomad_enabled
  source   = "./nomad"

  variables = {
    image        = resource.build.app.image
    network      = resource.network.onprem.id
    ingress_port = variable.nomad_ingress_port
  }
}

module "kubernetes" {
  disabled = !variable.kubernetes_enabled
  source   = "./kubernetes"

  variables = {
    image          = resource.build.app.image
    network        = resource.network.onprem.id
    ingress_port   = variable.kubernetes_ingress_port
    container_port = module.container.output.local_port
  }
}


output "KUBECONFIG" {
  disabled = !variable.kubernetes_enabled

  value = module.kubernetes.output.kubeconfig
}

output "container_app" {
  disabled = !variable.container_enabled

  value = "http://${module.container.output.local_address}:${module.container.output.local_port}"
}

output "nomad_app" {
  disabled = !variable.nomad_enabled

  value = "http://${module.nomad.output.local_address}:${module.nomad.output.local_port}"
}

output "kubernetes_app" {
  disabled = !variable.kubernetes_enabled

  value = "http://${module.kubernetes.output.local_address}:${module.kubernetes.output.local_port}"
}