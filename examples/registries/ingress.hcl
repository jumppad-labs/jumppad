resource "ingress" "noauth" {
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

resource "ingress" "auth" {
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

resource "ingress" "insecure" {
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

output "noauth_addr" {
  value = resource.ingress.noauth.local_address
}

output "auth_addr" {
  value = resource.ingress.auth.local_address
}

output "insecure_addr" {
  value = resource.ingress.insecure.local_address
}