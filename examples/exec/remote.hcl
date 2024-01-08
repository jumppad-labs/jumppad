resource "container" "alpine" {
  image {
    name = "alpine"
  }

  command = ["tail", "-f", "/dev/null"]
}

resource "exec" "in_container" {
  target = resource.container.alpine

  script = <<-EOF
  #!/bin/sh -e

  ls -las
  EOF
}

resource "exec" "standalone" {
  image {
    name = "alpine"
  }

  script = <<-EOF
  #!/bin/sh -e

  ls -las
  EOF
}