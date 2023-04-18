output "consul_http_addr" {
  value = "http://${resource.container.consul.fqdn}:8500"
}