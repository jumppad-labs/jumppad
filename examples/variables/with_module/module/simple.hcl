variable "mod_network" {
  default = "modulenetwork"
}

network "modulenetwork" {
  subnet = "10.6.0.0/16"
}

container "consul" {
  image   {
    name = "consul:1.6.1"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]
  
  network   {
    name = var.mod_network
  }
}
