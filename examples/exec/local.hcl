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

  # Add the output
  echo "$EXEC_OUTPUT" >> /tmp/output.var
  echo "exec=install" >> $EXEC_OUTPUT
  EOF

  timeout = "30s"
}

resource "exec" "run" {
  depends_on = ["resource.exec.install"]

  script = <<-EOF
  #!/bin/sh
  ${data("test")}/consul agent -dev
  
  # We will never get here as the previous command blocks
  echo "exec=run" >> $EXEC_OUTPUT
  EOF

  daemon = true
}

output "local_exec_install" {
  value = resource.exec.install.output.exec
}

//output "local_exec_run" {
//  value = resource.exec.run.output.exec
//}