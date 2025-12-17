variable "consul_version" {
  default = "1.22"
}

variable "envoy_version" {
  default = "1.18.4"
}

variable "number_of_nodes" {
  default     = 1
  description = "Controls the number of nodes for the Consul server"
}