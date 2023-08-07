resource "ingress" "fake_service_2" {
  port = 19091

  target {
    id         = resource.nomad_cluster.dev.id
    named_port = "http"

    config = {
      job   = "example_2"
      group = "fake_service"
      task  = "fake_service"
    }
  }
}

output "fake_service_addr" {
  value = resource.ingress.fake_service_2.address
}