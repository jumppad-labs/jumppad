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
cluster "default" {
  driver  = "k3s" // default
  version = "1.16.0"

  nodes = 3

  network = "network.k8s"
}

network "k8s" {
  subnet = "10.4.0.0/16"
}

// runs helm install
helm "consul" {
  cluster = "cluster.default"
  chart   = "${env("SHIPYARD_CONFIG")}/charts/consul"
  values  = "./consul-values"

  health_check {
    pods = ["component=server,app=consul", "component=client,app=consul"] // is the pod running and healthy
  }
}

// runs kubectl apply
k8s_config "dashboard" {
  cluster = k8s_cluster.default
  config  = "./k8s_dashboard"
}

// expose a port to the local machine
// ingress will create a localy routable dns consul.cluster.ingress.local
ingress "consul" {
  target  = cluster.default
  service = "consul-consul-server" //kubernetes service or nomad job

  ports {
    local  = 8500
    remote = 8500
    host   = 8500
  }
}

ingress "web" {
  target = container.web

  ports {
    local  = 9090
    remote = 9090
    host   = 9090
  }
}

/*
// copies a local docker image to the cluster
image "fake-service" {
  target      = k8s_cluster.default
  local_image = "nicholasjackson/fake-service:v07.7"
}
*/
