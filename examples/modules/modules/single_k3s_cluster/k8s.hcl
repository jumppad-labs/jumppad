resource "k8s_cluster" "k3s" {
  network {
    id = resource.network.cloud.resource_id
  }

  copy_image {
    name = "shipyardrun/connector:v0.1.0"
  }
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kubeconfig
}