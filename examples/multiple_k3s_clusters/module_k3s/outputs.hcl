output "consul_http_addr" {
  value = "http://${resource.ingress.consul_http.address}"
}