resource "random_number" "number" {
    minimum = 1
    maximum = 10
}

output "number" {
    value = resource.random_number.number.value
}

resource "random_id" "id" {
    byte_length = 4
}

output "id_hex" {
    value = resource.random_id.id.hex
}

output "id_dec" {
    value = resource.random_id.id.dec
}

resource "random_password" "password" {
    length = 32
}

output "password" {
    value = resource.random_password.password.value
}

resource "random_uuid" "uuid" {
}

output "uuid" {
    value = resource.random_uuid.uuid.value
}

resource "random_creature" "creature" {}

output "creature" {
    value = resource.random_creature.creature.value
}