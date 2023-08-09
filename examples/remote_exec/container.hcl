resource "network" "onprem" {
  subnet = "10.6.0.0/16"
}

resource "container" "alpine" {
  image {
    name = "alpine"
  }

  command = ["tail", "-f", "/dev/null"]

  network {
    id         = resource.network.onprem.id
    ip_address = "10.6.0.200" // optional
    aliases    = ["myalias"]
  }
}

resource "remote_exec" "in_container" {
  target = resource.container.alpine

  script = <<-EOF
  #/bin/sh -e

  ls -las
  EOF
}

resource "remote_exec" "standalone" {
  image {
    name = "alpine"
  }

  script = <<-EOF
  #/bin/sh -e

  ls -las
  EOF
}