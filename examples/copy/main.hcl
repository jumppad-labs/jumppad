resource "copy" "local" {
  source      = "${dir()}/files/foo"
  destination = "${data("copy")}/local/foo"
}

resource "copy" "local_relative" {
  source      = "./files/foo"
  destination = "${data("copy")}/local_relative"
}


resource "copy" "http" {
  source      = "https://www.foundanimals.org/wp-content/uploads/2023/02/twenty20_b4e89a76-af70-4567-b92a-9c3bbf335cb3.jpg"
  destination = "${data("copy")}/http"
}

resource "copy" "git" {
  source      = "github.com/jumppad-labs/examples"
  destination = "${data("copy")}/git"
}

resource "copy" "zip" {
  source      = "https://releases.hashicorp.com/nomad/1.6.3/nomad_1.6.3_linux_amd64.zip"
  destination = "${data("copy")}/zip"
}