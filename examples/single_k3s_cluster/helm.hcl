resource "helm" "consul" {
  cluster = resource.k8s_cluster.k3s.id

  # When no repositroy is specified, either a local path or go getter URL

  repository {
    name = "hashicorp"
    url  = "https://helm.releases.hashicorp.com"
  }

  chart   = "hashicorp/consul"
  version = "v0.40.0"

  values = "./helm/consul-values.yaml"

  health_check {
    timeout = "240s"
    pods = [
      "component=connect-injector",
      "component=client",
      "component=controller",
      "component=server",
    ]
  }
}

resource "helm" "vault" {
  cluster = resource.k8s_cluster.k3s.id

  repository {
    name = resource.helm.consul.repository.name // this also forces a dependency to be created
    url  = resource.helm.consul.repository.url  // vault will always be applied after consul
  }

  chart   = "hashicorp/vault" # When repository specified this is the name of the chart
  version = "v0.18.0"         # Version of the chart when repository specified

  values = "./helm/vault-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["app.kubernetes.io/name=vault"]
  }
}