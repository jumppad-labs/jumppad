resource "network" "main" {
  subnet = "10.6.0.0/16"
}

resource "docs" "docs" {
  network {
    id = resource.network.main.id
  }

  image {
    name = "ghcr.io/jumppad-labs/docs:v0.0.2"
  }

  content = [
    resource.book.terraform_basics.id
  ]
}

resource "book" "terraform_basics" {
  title = "Understanding Terraform basics"

  chapters = [
    resource.chapter.introduction.id,
  ]
}

resource "chapter" "introduction" {
  page "introduction" {
    title   = "Introduction"
    content = ""
  }
}