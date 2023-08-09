resource "random_number" "api_port" {
  minimum = 10000
  maximum = 20000
}

resource "k8s_cluster" "dev" {
  // use a random port for the cluster
  api_port = resource.random_number.api_port.value

  network {
    id = variable.network_id
  }
}

resource "helm" "consul" {
  cluster = resource.k8s_cluster.dev
  chart   = "github.com/hashicorp/consul-k8s?ref=${variable.consul_helm_version}//charts/consul"
  values  = "./helm/consul-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["release=consul"]
  }
}

resource "ingress" "consul_http" {
  port = variable.consul_port

  target {
    resource = resource.k8s_cluster.dev
    port     = 8500

    config = {
      service   = "consul-consul-server"
      namespace = "default"
    }
  }
}