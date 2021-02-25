data_dir = "/tmp"
log_level = "DEBUG"

datacenter = "dc1"
primary_datacenter = "dc1"

server = true

bootstrap_expect = 1
ui = true

bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"
advertise_addr = "10.6.0.200"

ports {
  grpc = 8502
}

connect {
  enabled = true
}
