network "onprem" {
  subnet = "10.5.0.0/16"
}

network "nomad" {
  subnet = "10.6.0.0/16"
}

network "k8s" {
  subnet = "10.4.0.0/16"
}

// wan network is automatically created
// subnet = "192.168.0.0/16"
