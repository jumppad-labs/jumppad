resource "remote_exec" "nomad_jobs" {
  depends_on = ["resource.nomad_cluster.dev"]

  target  = resource.container.tools.id
  command = ["nomad", "run", "/files/example.nomad"]
}