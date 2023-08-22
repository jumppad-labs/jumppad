variable "image" {
  default = ""
}

variable "network" {
  default = ""
}

variable "ingress_port" {
  default = 0
}

variable "default_port" {
  default = 19090
}

resource "random_number" "port" {
  minimum = 10000
  maximum = 20000
}

resource "nomad_cluster" "dev" {
  client_nodes = 2

  network {
    id = variable.network
  }

  copy_image {
    name = variable.image
  }
}

resource "template" "app_job" {
  source = <<-EOF
 job "app" {
   datacenters = ["dc1"]
   type = "service"
   
   group "app" {
     count = 1
 
     network {
       port  "http" { 
         to = 9090
         static = 9090
       }
     }
 
     task "app" {
       driver = "docker"
 
       config {
         image = "${variable.image}"
         ports = ["http"]
       }
     }
   }
 }
 EOF

  destination = "${data("jobs")}/job.nomad"
}

resource "nomad_job" "app" {
  cluster = resource.nomad_cluster.dev

  paths = [resource.template.app_job.destination]
}

resource "ingress" "app" {
  port = variable.ingress_port == 0 ? resource.random_number.port.value : variable.default_port

  target {
    resource   = resource.nomad_cluster.dev
    named_port = "http"

    config = {
      job   = "app"
      group = "app"
      task  = "app"
    }
  }
}


output "local_address" {
  value = "localhost"
}

output "local_port" {
  value = variable.ingress_port == 0 ? resource.random_number.port.value : variable.default_port
}