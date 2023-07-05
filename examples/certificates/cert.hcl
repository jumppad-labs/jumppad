resource "certificate_ca" "root" {
  output = data("certs")
}

resource "certificate_leaf" "nomad" {
  ca_key  = resource.certificate_ca.root.private_key.path
  ca_cert = resource.certificate_ca.root.certificate.path

  ip_addresses = ["127.0.0.1"]

  dns_names = [
    "localhost",
    "localhost:30090",
    "30090",
    "connector",
    "connector",
  ]

  output = data("certs")
}