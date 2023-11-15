resource "exec" "install" {
  script = <<-EOF
  #!/bin/sh
  cat <<EOT > /tmp/exec
  #!/bin/sh
  while true
  do
    echo "hello world"
    sleep 1
  done
  EOT
  chmod +x /tmp/exec
  EOF
}

resource "exec" "run" {
  depends_on = ["resource.exec.install"]

  script = <<-EOF
  #!/bin/sh
  /tmp/exec
  EOF

  daemon = true
}