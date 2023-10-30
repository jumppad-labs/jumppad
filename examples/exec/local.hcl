resource "exec" "install" {
  script = <<-EOF
  #!/bin/sh
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')

  if [ ! -f /tmp/consul ]; then
    curl -L -o /tmp/consul.zip https://releases.hashicorp.com/consul/1.16.2/consul_1.16.2_$${OS}_$${ARCH}.zip
    cd /tmp && unzip ./consul.zip
  fi
  EOF
}

resource "exec" "run" {
  depends_on = ["resource.exec.install"]

  script = <<-EOF
  #!/bin/sh
  /tmp/consul agent -dev
  EOF

  daemon = true
}