container "tools" {
  image   {
    name = "shipyardrun/tools:latest"
  }

  command = ["tail", "-f", "/dev/null"]

 # Setup files 
  volume {
    source      = "./app_config"
    destination = "/files"
  }

  network {
    name = "network.cloud"
  }
  
  env {
    key = "NOMAD_ADDR"
    value = "http://server.dev.nomad_cluster.shipyard.run:4646"
  }
}