resource "random_pet" "pet" {}

variable "first" {}
variable "second" {}
variable "third" {}

output "first" {
  value = var.first
}

output "second" {
  value = var.second
}

output "third" {
  value = var.third
}