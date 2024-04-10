# requires example plugin github.com/jumppad-labs/plugin-sdk/example to be 
# built and installed to $HOME/.jumppad/plugins

resource "example" "nice" {
  value = "Erik"
}

output "erik_is_nice" {
  value = resource.example.nice.value
}