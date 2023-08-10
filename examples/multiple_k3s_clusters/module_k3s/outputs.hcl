output "k8s_port" {
  value = resource.random_number.api_port.value
}

output "consul_http_addr" {
  value = "http://${resource.ingress.consul_http.address}"
}