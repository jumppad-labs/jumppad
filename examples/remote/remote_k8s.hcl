container "remote_k8s" {
  image {
    name = "terraform"
  }

  command = "terraform apply -auto-approve"

  volume {
    source      = "${data("/terraform/.state/")}"
    destination = "/terraform"
  }

  volume {
    source      = "./modules/k8s_doks"
    destination = "/terraform"
  }

  volume {
    source      = "${shipyard()}/.config/doks"
    destination = "/terraform/kubeconfig.yaml"
  }
}

container "local_connector" {
  image {
    name = "ghcr.io/jumppad-labs/connector:v0.4.0"
  }

  env_var = {
    "BIND_ADDR_GRPC" : "0.0.0.0:9090"
    "BIND_ADDR_HTTP" : "0.0.0.0:9091"
    "LOG_LEVEL" : "debug"
  }

  port_range {
    range       = "9090-9091"
    enable_host = true
  }

  port_range {
    range       = "12000-12100"
    enable_host = true
  }

  network {
    name = "network.cloud"
  }
}

# In module
# 1. Create remote cluster
# 2. Deploy connector with external LB 
# 3. Run script as exect to configure connector
# 4. Deploy application to K8s 
# 4. Run fake-service locally 
# 5. Test