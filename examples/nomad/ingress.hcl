resource "ingress" "fake_service_1" {
  port = 19090

  target {
    id         = resource.nomad_cluster.dev.id
    named_port = "http"

    config = {
      job   = "example_1"
      group = "fake_service"
      task  = "fake_service"
    }
  }
}

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

output "fake_service_addr_1" {
  value = resource.ingress.fake_service_1.address
}

output "fake_service_addr_2" {
  value = resource.ingress.fake_service_2.address
}