resource "container" "alpine" {
  image {
    name = "alpine"
  }

  command = ["tail", "-f", "/dev/null"]

  volume {
    source      = data("test")
    destination = "/data"
  }
}

resource "exec" "in_container" {
  target = resource.container.alpine

  script = <<-EOF
  #!/bin/sh -e

  touch /data/container.txt
  EOF
}

resource "exec" "standalone" {
  image {
    name = "alpine"
  }

  script = <<-EOF
  #!/bin/sh -e

  touch /data/standalone.txt
  EOF

  volume {
    source      = data("test")
    destination = "/data"
  }
}