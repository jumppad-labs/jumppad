variable "version" {
  default = "consul:1.10.6"
}

resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}

variable "test_var" {
  default = [1, 2]
}

resource "container" "consul" {
  image {
    name = variable.version
  }

  command = ["consul", "agent", "-dev", "-client", "0.0.0.0"]

  network {
    id         = resource.network.onprem.meta.id
    ip_address = "10.6.0.200"
  }

  environment = {
    file_dir              = dir()
    env                   = env("HOME")
    home                  = home()
    file                  = file("./default.vars")
    data                  = data("mine")
    data_with_permissions = data_with_permissions("mine", 755)
    docker_ip             = docker_ip()
    docker_host           = docker_host()
    var_len               = len(variable.test_var)
    os                    = system("os")
    arch                  = system("arch")
    exists                = exists("file")
  }
}
