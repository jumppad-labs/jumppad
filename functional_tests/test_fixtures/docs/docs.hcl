docs "docs" {
  path  = "./docs"
  port  = 8080

  network {
    name = "network.docs"
  }

  index_title = "Test"
  index_pages = ["index", "other"]
}