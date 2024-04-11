resource "exec" "install" {
  script = <<-EOF
  #!/bin/sh
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')

  if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
  fi

  curl -L -o ${data("test")}/consul.zip https://releases.hashicorp.com/consul/1.16.2/consul_1.16.2_$${OS}_$${ARCH}.zip
  cd ${data("test")} && unzip ./consul.zip

  echo "TEST=
  EOF

  timeout = "30s"
}

resource "exec" "run" {
  depends_on = ["resource.exec.install"]

  script = <<-EOF
  #!/bin/sh
  ${data("test")}/consul agent -dev
  EOF

  daemon = true
}