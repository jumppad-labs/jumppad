resource "template" "docker_registry" {
  source = <<-EOF
  {
    "proxies": {
      "http-proxy": "http://default.image-cache.jumppad.dev:3128",
      "https-proxy": "http://default.image-cache.jumppad.dev:3128",
      "no-proxy": "insecure.container.jumppad.dev"
    },
    "insecure-registries" : [ "insecure.container.jumppad.dev:5003" ]
  }
  EOF

  destination = "${data("registry")}/daemon.json"
}

resource "nomad_cluster" "dev" {
  client_nodes = 1

  datacenter = "dc1"

  network {
    id = resource.network.cloud.id
  }

  volume {
    source      = resource.template.docker_registry.destination
    destination = "/etc/docker/daemon.json"
  }
}
