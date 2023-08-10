variable "image" {
  default = ""
}

variable "network" {
  default = ""
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
  port = 19090

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