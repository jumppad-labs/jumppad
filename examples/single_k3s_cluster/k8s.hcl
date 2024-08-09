resource "k8s_cluster" "k3s" {
  network {
    id = resource.network.cloud.meta.id
  }

  copy_image {
    name = "ghcr.io/jumppad-labs/connector:v0.4.0"
  }
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kube_config.path
}