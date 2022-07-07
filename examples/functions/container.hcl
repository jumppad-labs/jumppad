variable "version" {
  default = "consul:1.10.6"
}

network "onprem" {
  subnet = "10.6.0.0/16"
}

variable "test_var" {
  default = [1, 2]
}

container "consul" {
  image {
    name = var.version
  }

  command = ["consul", "agent", "-dev", "-client", "0.0.0.0"]

  network {
    name       = "network.onprem"
    ip_address = "10.6.0.200"
  }

  env_var = {
    file_dir          = file_dir()
    env               = env("HOME")
    k8s_config        = k8s_config("dc1")
    k8s_config_docker = k8s_config_docker("dc1")
    home              = home()
    shipyard          = shipyard()
    file              = file("./default.vars")
    data              = data("mine")
    docker_ip         = docker_ip()
    docker_host       = docker_host()
    shipyard_ip       = shipyard_ip()
    cluster_api       = cluster_api("nomad_cluster.dc1")
    var_len           = len(var.test_var)
  }
}
