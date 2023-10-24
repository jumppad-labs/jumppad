resource "vm" "x86_64" {
	config {
		arch = "x86_64" // default -> host arch
	}

  resources {
    cpu = 2
    memory = 2048 // mb
  }

  // disk {
  //   type = "raw"
  //   source = "/Users/erik/code/instruqt/qemu/debian-live.iso"
  // }

  disk {
    type = "qcow2"
    source = "/Users/erik/code/instruqt/qemu/cloud.img"
  }

  // disk "name" {
  //   type = "ext4"
  //   size = 100 // mb
  //   source = "/path/on/host"
  //   boot_order = 1
  // }

  volume {
    source = "/tmp/test"
    destination = "/tmp/test"
  }

  // network {
  //   id = resource.network.main.id
  //   ip_address = "10.0.10.5"
  // }

  port {
    local  = 22
    host   = 2201
  }

  vnc {
    port = 8999
  }

  cloud_init {
    network_config = <<-EOF
    version: 2
    ethernets:
      ens2:
        dhcp4: true
    EOF

    meta_data = <<-EOF
    instance-id: instruqt
    local-hostname: instruqt
    EOF

    user_data = <<-EOF
    #cloud-config
    password: password
    chpasswd:
      expire: False
    debug: True
    disable_root: False
    ssh_deletekeys: False
    ssh_pwauth: True
    ssh_authorized_keys:
      - ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEsCSbX1+LRRh8ClnXl2/RLXE1CpJgJ2j9RZNJbwKSDM
    EOF
  }
}

// resource "vm" "aarch64" {
// 	config {
// 		arch = "aarch64" //"x86_64" // default -> host arch
// 	}

//   resources {
//     cpu = 2
//     memory = 2048 // mb
//   }

//   disk {
//     type = "qcow2"
//     source = "/Users/erik/code/instruqt/qemu/ubuntu.qcow2"
//   }

//   //  disk {
//   //   type = "raw"
//   //   source = "/Users/erik/code/instruqt/qemu/ubuntu.iso"
//   // }

//   // disk "name" {
//   //   type = "ext4"
//   //   size = 100 // mb
//   //   source = "/path/on/host"
//   //   boot_order = 1
//   // }

//   volume {
//     source = "/tmp/test"
//     destination = "/tmp/test"
//   }

//   // network {
//   //   id = resource.network.main.id
//   //   ip_address = "10.0.10.5"
//   // }

//   port {
//     local  = 22
//     host   = 2202
//   }

//   vnc {
//     port = 8902
//   }
// }