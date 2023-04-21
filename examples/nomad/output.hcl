output "NOMAD_ADDR" {
  value = "http://${resource.nomad_cluster.dev.external_ip}:${resource.nomad_cluster.dev.api_port}"
}
