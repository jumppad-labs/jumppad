resource "copy" "local" {
  source      = "${dir()}/files/foo"
  destination = "${data("copy")}/local/foo"
}

resource "copy" "local_relative" {
  source      = "./files/foo"
  destination = "${data("copy")}/local_relative"
}


resource "copy" "http" {
  source      = "https://cdn.sanity.io/images/fhoo4r9z/production/50bf44d383cccc9f35cf1209844f72e8e1cfb799-400x400.jpg?w=285&h=285&q=85&auto=format"
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