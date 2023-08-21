variable "container_enabled" {
  default = true
}

variable "nomad_enabled" {
  default = true
}

variable "kubernetes_enabled" {
  default = true
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
    image   = resource.build.app.image
    network = resource.network.onprem.id
  }
}

module "nomad" {
  disabled = !variable.nomad_enabled
  source   = "./nomad"

  variables = {
    image   = resource.build.app.image
    network = resource.network.onprem.id
  }
}

module "kubernetes" {
  disabled = !variable.kubernetes_enabled
  source   = "./kubernetes"

  variables = {
    image   = resource.build.app.image
    network = resource.network.onprem.id
  }
}

// exposes a local service running at port 9090
// and creates a kubernetes service fake-service.jumppad.svc:9090
resource "ingress" "local_app_to_k8s" {
  disabled = !variable.kubernetes_enabled

  port         = module.container.output.local_port
  expose_local = true

  target {
    resource = module.kubernetes.output.cluster
    port     = 9090

    config = {
      service = "fake-service"
    }
  }
}

output "KUBECONFIG" {
  disabled = !variable.kubernetes_enabled

  value = module.kubernetes.output.kubeconfig
}