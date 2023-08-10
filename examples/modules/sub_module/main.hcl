variable "version" {
  default = ""
}


module "consul" {
  source = "../../single_file"

  variables = {
    version    = "consul:${variable.version}"
    port_range = "18502-18504"
  }
}
