output "consul_http_addr" {
  value = "http://${resource.container.consul.fqrn}:8500"
}