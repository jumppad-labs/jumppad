variable "nomad_enabled" {
  default = true
}

variable "k8s_enabled" {
  default = false
}

variable "auth_ip_address" {
  default = "10.6.0.183"
}

variable "noauth_ip_address" {
  default = "10.6.0.184"
}

variable "insecure_ip_address" {
  default = "10.6.0.185"
}