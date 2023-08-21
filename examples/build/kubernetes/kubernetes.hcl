variable "image" {
  default = ""
}

variable "network" {
  default = ""
}

resource "k8s_cluster" "k3s" {
  network {
    id = variable.network
  }

  copy_image {
    name = variable.image
  }
}

output "kubeconfig" {
  value = resource.k8s_cluster.k3s.kubeconfig
}

output "cluster" {
  value = resource.k8s_cluster.k3s
}