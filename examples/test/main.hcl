variable "storage" {
  default = "128Mb"
}

resource "network" "local" {
  subnet = "10.6.0.0/16"
}

resource "k8s_cluster" "dev" {
  network {
    id = resource.network.local.id
  }
}

resource "template" "helm_values" {
  source = <<-EOF
   ---
   server:
     dataStorage:
       size: ${variable.storage} 
     dev:
       enabled: true
     standalone:
       enabled: true
     authDelegator:
       enabled: true
   ui:
     enabled: true
  EOF

  destination = "${data("helm-values")}/default-values.yaml"
}

resource "helm" "vault" {
  cluster          = resource.k8s_cluster.dev
  namespace        = "vault"
  create_namespace = true

  repository {
    name = "hashicorp"
    url  = "https://helm.releases.hashicorp.com"
  }

  chart   = "hashicorp/vault"
  version = "0.24.0"

  values = resource.template.helm_values.destination

  health_check {
    timeout = "120s"
    pods    = ["app.kubernetes.io/name=vault", "app.kubernetes.io/name=vault-agent-injector"]
  }
}

resource "ingress" "vault_http" {
  port = 18200

  target {
    resource = resource.k8s_cluster.dev
    port     = 8200

    config = {
      service   = "vault-ui"
      namespace = "vault"
    }
  }
}

output "VAULT_ADDR" {
  value = "http://${resource.ingress.vault_http.local_address}"
}

output "VAULT_TOKEN" {
  value = "root"
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.dev.kubeconfig
}