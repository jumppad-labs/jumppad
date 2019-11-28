/* k8s on k3s
- kubernetes cluster using k3s
- consul installed on cluster using helm chart
- kubernetes configuration such as dashboard, pvc, etc.
- vscode in environment
- tools container pointing at cluster
- exposing ports from cluster to host machine
- networks to add the cluster/loadbalancer to
*/

/* nomad
- nomad server
- consul installed on cluster using nomad
- vscode in environment
- tools container pointing at cluster
- exposing ports from cluster to host machine
- networks to add the cluster/loadbalancer to
*/

/* multi-cluster stack
- nomad server
- consul installed on cluster using nomad
- kubernetes cluster using k3s
- consul installed on cluster using helm chart
- kubernetes configuration such as dashboard, pvc, etc.
- vscode in environment
- tools container pointing at cluster
- exposing ports from cluster to host machine
- networks to add the clusters/loadbalancer to
*/

// k3d create
k8s_cluster "default" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 3

  network {
    name = network.wan.name
  }
}

// runs helm install
k8s_helm "consul" {
  cluster = k8s_cluster.default
  values  = "./consul-values"
}

// runs kubectl apply
k8s_config "dashboard" {
  cluster = k8s_cluster.default
  config  = "./k8s_dashboard"
}

nomad_cluster "default" {
  version = "0.10.0"

  network = network.wan
}

nomad_job "consul" {
  cluster = nomad_cluster.default
  config  = "./consul"
}

// runs docker network create
network "wan" {
  subnet = "192.168.0.0/16"
}

// run adhoc docker containers
container "web" {
  image = "consul:1.6.2"

  network {
    name       = network.wan.name
    ip_address = "192.168.0.1" // if ommited will auto assign
  }
}

yard apply -f step1.hcl
yard apply -f step2.hcl
yard delete -f step2.hcl

nomad_ingress "nomad" {}

container_ingress "web" {}

// expose a port to the local machine
// ingress will create a localy routable dns consul.cluster.ingress.local
ingress "consul-k8s" {
  target    = k8s_cluster.default
  service   = "consul-consul-server" //kubernetes service or nomad job

  ports {
    local  = 8500
    remote = 8500
  }

  ports {
    local  = 8600
    remote = 8600
  }

  bind_host = true // should the ports be bound to the host machine on 0.0.0.0

  network {
    name       = network.wan.name
    ip_address = "192.168.0.1" // if ommited will auto assign
  }
}

ingress "web" {
  target = container.web

  local_addr = "0.0.0.0"

  ports {
    local  = 9090
    remote = 9090
  }

  ports {
    local  = 8600
    remote = 8600
  }

  network {
    name       = network.wan.name
    ip_address = "192.168.0.1" // if ommited will auto assign
  }
}

// copies a local docker image to the cluster
image "fake-service" {
  target = k8s_cluster.default
  local_image = "nicholasjackson/fake-service:v07.7"
}