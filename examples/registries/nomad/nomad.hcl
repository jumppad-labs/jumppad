variable "network_id" {
  default = ""
}

resource "nomad_cluster" "dev" {
  client_nodes = 1

  datacenter = "dc1"

  network {
    id = variable.network_id
  }

  // add configuration to allow cache bypass and insecure registry
  config {
    docker {
      no_proxy            = ["insecure.container.local.jmpd.in"]
      insecure_registries = ["insecure.container.local.jmpd.in:5003"]
    }
  }
}

resource "nomad_job" "noauth" {
  cluster = resource.nomad_cluster.dev

  paths = ["./files/noauth.nomad"]

  health_check {
    timeout = "60s"
    jobs    = ["noauth"]
  }
}

resource "nomad_job" "auth" {
  cluster = resource.nomad_cluster.dev

  paths = ["./files/auth.nomad"]

  health_check {
    timeout = "60s"
    jobs    = ["auth"]
  }
}

resource "nomad_job" "insecure" {
  cluster = resource.nomad_cluster.dev

  paths = ["./files/insecure.nomad"]

  health_check {
    timeout = "60s"
    jobs    = ["insecure"]
  }
}

resource "ingress" "nomad_noauth" {
  port = 19090

  target {
    resource   = resource.nomad_cluster.dev
    named_port = "http"

    config = {
      job   = "noauth"
      group = "app"
      task  = "app"
    }
  }
}

resource "ingress" "nomad_auth" {
  port = 19091

  target {
    resource   = resource.nomad_cluster.dev
    named_port = "http"

    config = {
      job   = "auth"
      group = "app"
      task  = "app"
    }
  }
}

resource "ingress" "nomad_insecure" {
  port = 19092

  target {
    resource   = resource.nomad_cluster.dev
    named_port = "http"

    config = {
      job   = "insecure"
      group = "app"
      task  = "app"
    }
  }
}

output "nomad_noauth_addr" {
  value = resource.ingress.nomad_noauth.local_address
}

output "nomad_auth_addr" {
  value = resource.ingress.nomad_auth.local_address
}

output "nomad_insecure_addr" {
  value = resource.ingress.nomad_insecure.local_address
}