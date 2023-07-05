output "consul_http_addr" {
  description = "HTTP address for the consul server"
  value       = "http://${resource.container.consul.fqrn}:8500"
}