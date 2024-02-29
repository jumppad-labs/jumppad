variable "network_id" {
  default = ""
}

resource "k8s_cluster" "k3s" {
  network {
    id = variable.network_id
  }

  // add configuration to allow cache bypass and insecure registry
  config {
    docker {
      no_proxy            = ["insecure.container.local.jumpd.in"]
      insecure_registries = ["insecure.container.local.jmpd.in:5003"]
    }
  }
}

resource "k8s_config" "noauth" {
  cluster = resource.k8s_cluster.k3s

  paths = [
    "./files/noauth.yaml",
  ]

  wait_until_ready = true
}

resource "k8s_config" "auth" {
  cluster = resource.k8s_cluster.k3s

  paths = [
    "./files/auth.yaml",
  ]

  wait_until_ready = true
}

resource "k8s_config" "insecure" {
  cluster = resource.k8s_cluster.k3s

  paths = [
    "./files/insecure.yaml",
  ]

  wait_until_ready = true
}

resource "ingress" "k8s_noauth" {
  port = 29090

  target {
    resource = resource.k8s_cluster.k3s
    port     = 19090

    config = {
      service   = "noauth"
      namespace = "default"
    }
  }
}

resource "ingress" "k8s_auth" {
  port = 29091

  target {
    resource = resource.k8s_cluster.k3s
    port     = 19091

    config = {
      service   = "auth"
      namespace = "default"
    }
  }
}

resource "ingress" "k8s_insecure" {
  port = 29092

  target {
    resource = resource.k8s_cluster.k3s
    port     = 19092

    config = {
      service   = "insecure"
      namespace = "default"
    }
  }
}

output "k8s_noauth_addr" {
  value = resource.ingress.k8s_noauth.local_address
}

output "k8s_auth_addr" {
  value = resource.ingress.k8s_auth.local_address
}

output "k8s_insecure_addr" {
  value = resource.ingress.k8s_insecure.local_address
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kubeconfig
}