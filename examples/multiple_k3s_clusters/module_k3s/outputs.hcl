output "consul_http_addr" {
  value = "http://${resource.ingress.consul-http.address}"
}