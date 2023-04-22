resource "container" "tools" {
  image {
    name = "shipyardrun/tools:latest"
  }

  command = ["tail", "-f", "/dev/null"]

  # Setup files 
  volume {
    source      = "./app_config"
    destination = "/files"
  }

  network {
    name = resource.network.cloud.id
  }

  environment = {
    "NOMAD_ADDR" : "http://server.dev.nomad-cluster.shipyard.run:4646"
  }
}