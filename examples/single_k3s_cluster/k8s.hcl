resource "k8s_cluster" "k3s" {
  driver = "k3s" // default

  nodes = 1 // default

  network {
    id = resource.network.cloud.id
  }

  copy_image {
    name = "shipyardrun/connector:v0.1.0"
  }
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kubeconfig
}