variable "container_enabled" {
  default = true
}

variable "nomad_enabled" {
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

// expose the local app via https
//ingress "local_build" {
//  // enable https for the application
//  // at app.local.jumppad.dev
//  https {
//    host = "app"
//  }
//
//  target {
//    id = module.container.output.app_id
//  }
//}
//
//ingress "local_app_in_nomad" {
//  // enable 
//  // at app.local.jumppad.dev
//  https {
//    host = "app"
//  }
//
//  target {
//    id = module.container.output.app_id
//  }
//
//}