resource "docs" "docs" {
  image {
    name = "ghcr.io/jumppad-labs/docs:v0.0.1"
  }
  path            = "./docs"
  navigation_file = "./config/navigation.jsx"
  port            = 8080
}