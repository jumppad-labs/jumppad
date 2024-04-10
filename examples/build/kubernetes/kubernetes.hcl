variable "image" {
  default = ""
}

variable "network" {
  default = ""
}

variable "ingress_port" {
  default = 0
}

variable "default_port" {
  default = 19090
}

// port where the build container is running
variable "container_port" {
  default = 0
}

resource "k8s_cluster" "dev" {
  network {
    id = variable.network
  }

  copy_image {
    name = variable.image
  }
}

resource "random_number" "port" {
  minimum = 10000
  maximum = 20000
}

// exposes a local service running at port 9090
// and creates a kubernetes service fake-service.jumppad.svc:9090
resource "ingress" "local_app_to_k8s" {
  port         = variable.container_port
  expose_local = true

  target {
    resource = resource.k8s_cluster.dev
    port     = 9090

    config = {
      service = "local-app"
    }
  }
}

resource "template" "app_job" {
  source = <<-EOF
  apiVersion: v1
  kind: Service
  metadata:
    name: app
  spec:
    selector:
      app: app
    ports:
      - protocol: TCP
        port: 9090
        targetPort: 9090

  ---
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: build-deployment
    labels:
      app: app
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: app
    template:
      metadata:
        labels:
          app: app
      spec:
        containers:
        - name: app
          image: ${variable.image}
          ports:
          - containerPort: 9090
          env:
          - name: "UPSTREAM_URL"
            value: "http://${resource.ingress.local_app_to_k8s.remote_address}"
  EOF

  destination = "${data("jobs")}/app.yaml"
}

resource "k8s_config" "app" {
  cluster = resource.k8s_cluster.dev

  paths = [
    resource.template.app_job.destination
  ]

  wait_until_ready = true
}

resource "ingress" "k8s_app" {
  port = variable.ingress_port == 0 ? resource.random_number.port.value : variable.default_port

  target {
    resource = resource.k8s_cluster.dev
    port     = 9090

    config = {
      service   = "app"
      namespace = "default"
    }
  }
}

output "kubeconfig" {
  value = resource.k8s_cluster.dev.kube_config.path
}

output "cluster" {
  value = resource.k8s_cluster.dev
}

output "local_address" {
  value = "localhost"
}

output "local_port" {
  value = variable.ingress_port == 0 ? resource.random_number.port.value : variable.default_port
}