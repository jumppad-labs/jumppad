module "consul" {
	source = "./module"
}

variable "network" {
  default = "onprem"
  description = "Name of the default network"
}
