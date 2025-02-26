resource "exec" "local" {
  script = <<-EOF
  #!/bin/bash
  echo "key=value" >> $EXEC_OUTPUT
  EOF
}

# resource "container" "test" {
#   image = "alpine:latest"
#   command = ["sh", "-c", "echo key=value >> $EXEC_OUTPUT"]
# }

# bla

# "container" "test" {
#   image = "alpine:latest"
#   command = ["sh", "-c", "echo key=value >> $EXEC_OUTPUT"]
# }

output "works" {
  value = resource.exec.local.output.key
}

output "broken" {
  value = resource.exec.local
}

output "empty" {}