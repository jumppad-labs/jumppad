resource "vm" "test2" {
	config {
		arch = "x86_64" // default -> host arch
	}

  image = "/Users/erik/code/jumppad/snapshot-restore/images/custom.qcow2" 

  resources {
    cpu = 2
    memory = 1024 // mb
  }

  // disk "name" {
  //   type = "ext4"
  //   size = 100 // mb
  // }

  // volume {
  //   source = "/path/on/host"
  //   destination = "/path/in/vm"
  // }

  // network {
  //   id = resource.network.main.id
  //   ip_address = "10.0.10.5"
  // }

  // port {
  //   local  = 8000
  //   remote = 8000
  //   host   = 8000
  // }

  // cloud_config = <<-EOF
  // runcmd: |-
  //   apt update
  //   apt install -y curl
  // EOF
}