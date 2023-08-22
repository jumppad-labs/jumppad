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

resource "template" "app_job" {
  source = <<-EOF
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: build-deployment
    labels:
      app: build
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: build
    template:
      metadata:
        labels:
          app: build
      spec:
        containers:
        - name: build
          image: ${variable.image}
          ports:
          - containerPort: 9090
  EOF

  destination = "${data("jobs")}/app.yaml"
}

resource "k8s_config" "app" {
  cluster = resource.k8s_cluster.k3s

  paths = [
    resource.template.app_job.destination
  ]

  wait_until_ready = true
}

output "kubeconfig" {
  value = resource.k8s_cluster.k3s.kubeconfig
}

output "cluster" {
  value = resource.k8s_cluster.k3s
}