resource "ingress" "consul-http" {
  source {
    driver = "local"

    config {
      port = 18500
    }
  }

  destination {
    driver = "k8s"

    config {
      cluster = resource.k8s_cluster.k3s.id
      address = "consul-consul-server.default.svc"
      port    = 8500
    }
  }
}

resource "ingress" "consul-lan" {
  source {
    driver = "local"

    config {
      port = 8300
    }
  }

  destination {
    driver = "k8s"

    config {
      cluster = resource.k8s_cluster.k3s.id
      address = "consul-consul-server.default.svc"
      port    = 8300
    }
  }
}

resource "ingress" "vault-http" {
  source {
    driver = "local"

    config {
      port = 18200
    }
  }

  destination {
    driver = "k8s"

    config {
      cluster = resource.k8s_cluster.k3s.id
      address = "vault.default.svc"
      port    = 8200
    }
  }
}
