resource "vm" "example" {
  kernel = "/home/erik/code/jumppad/cloudhypervisor-go-sdk/examples/files/vmlinuz"
  boot_args = "root=/dev/vda1 ro console=tty1 console=ttyS0"
  initrd = "/home/erik/code/jumppad/cloudhypervisor-go-sdk/examples/files/initrd"

  disk {
    path = "/home/erik/code/jumppad/cloudhypervisor-go-sdk/examples/files/noble.raw"
  }

  serial = "/tmp/serial"
}