resource "container" "tools" {
  image {
    name = "shipyardrun/hashicorp-tools:v0.7.0"
  }

  command = ["tail", "-f", "/dev/null"]

  # Setup files 
  volume {
    source      = "./app_config"
    destination = "/files"
  }

  network {
    id = resource.network.cloud.id
  }

  environment = {
    "NOMAD_ADDR" = "http://${resource.nomad_cluster.dev.server_fqdn}:${resource.nomad_cluster.dev.api_port}"
  }
}