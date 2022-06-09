output "NOMAD_HTTP_ADDR" {
  value = cluster_api("nomad_cluster.dev")
}

output "NOMAD_ADDR" {
  value = cluster_api("nomad_cluster.dev")
}
