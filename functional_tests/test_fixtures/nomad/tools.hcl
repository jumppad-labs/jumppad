container "tools" {
  image   {
    name = "shipyardrun/tools:latest"
  }

  command = ["tail", "-f", "/dev/null"]

  # Shipyard config for Kube 
  volume {
    source      = "${env("HOME")}/.shipyard"
    destination = "/root/.shipyard"
  }

 # Setup files 
  volume {
    source      = "./app_config"
    destination = "/files"
  }

  network = "network.cloud"

  env {
    key = "VAULT_TOKEN"
    value = "root"
  }
  
  env {
    key = "KUBECONFIG"
    value = "/root/.shipyard/config/k3s/kubeconfig-docker.yaml"
  }
  
  env {
    key = "VAULT_ADDR"
    value = "http://vault-http.cloud.shipyard:8200"
  }
  
  env {
    key = "CONSUL_HTTP_ADDR"
    value = "http://consul-http.cloud.shipyard:8200"
  }
  
  env {
    key = "NOMAD_ADDR"
    value = "http://server.nomad.cloud.shipyard:4646"
  }
}