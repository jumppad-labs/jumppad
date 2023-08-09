output "consul_http_addr" {
  description = "HTTP address for the consul server"
  value       = "http://${resource.container.consul.container_name}:8500"
}