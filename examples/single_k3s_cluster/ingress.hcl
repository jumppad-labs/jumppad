resource "ingress" "consul_http" {
  port = 18500

  target {
    resource = resource.k8s_cluster.k3s
    port     = 8500

    config = {
      service   = "consul-consul-server"
      namespace = "default"
    }
  }
}

resource "ingress" "consul_lan" {
  port = 8300

  target {
    resource = resource.k8s_cluster.k3s
    port     = 8300

    config = {
      service   = "consul-consul-server"
      namespace = "default"
    }
  }
}

resource "ingress" "vault_http" {
  port = 18200

  target {
    resource = resource.k8s_cluster.k3s
    port     = 8200

    config = {
      service   = "vault"
      namespace = "default"
    }
  }
}

output "CONSUL_HTTP_ADDR" {
  value = resource.ingress.consul_http.local_address
}

output "VAULT_ADDR" {
  value = resource.ingress.vault_http.local_address
}