resource "build" "app" {
  container {
    dockerfile = "Dockerfile"
    context    = "./src"
  }
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}

resource "container" "app" {
  image {
    name = resource.build.app.image
  }

  command = ["/bin/app"]

  port {
    local  = 9090
    remote = 9090
    host   = 9090
  }

  network {
    id = resource.network.onprem.id
  }
}

resource "nomad_cluster" "dev" {
  client_nodes = 2

  network {
    id = resource.network.onprem.id
  }

  copy_image {
    name = resource.build.app.image
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
         image = "${resource.build.app.image}"
         ports = ["http"]
       }
     }
   }
 }
 EOF

  destination = "${data("jobs")}/job.nomad"
}

resource "nomad_job" "app" {
  cluster = resource.nomad_cluster.dev.id

  paths = [resource.template.app_job.destination]
}