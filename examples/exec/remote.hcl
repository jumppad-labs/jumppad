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

  echo "exec=container" >> $EXEC_OUTPUT
  EOF
}

resource "exec" "standalone" {
  image {
    name = "alpine"
  }

  script = <<-EOF
  #!/bin/sh -e

  touch /data/standalone.txt
  
  echo "exec=standalone" >> $EXEC_OUTPUT
  EOF

  volume {
    source      = data("test")
    destination = "/data"
  }
}

output "remote_exec_container" {
  value = resource.exec.in_container.output.exec
}

output "remote_exec_standalone" {
  value = resource.exec.standalone.output.exec
}