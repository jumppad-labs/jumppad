variable "network" {
  default = "onprem"
  description = "Name of the default network"
}

network "cloud" {
  subnet = "10.7.0.0/16"
}

network "onprem" {
  subnet = "10.6.0.0/16"
}

container "consul" {
  image   {
    name = "consul:1.6.1"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]
  
  network   {
    name = var.network
  }
}
