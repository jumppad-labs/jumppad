resource "jumppad" "config" {
  version = "f-plugins"

  plugin "alias" {
    source  = "github.com/jumppad-labs/example-plugin"
    local   = "./examples/plugins/example"
    version = "v0.1.0"
  }
}