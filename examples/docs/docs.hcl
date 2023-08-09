resource "docs" "docs" {
  content = [
    resource.book.terraform_basics
  ]
}

resource "book" "terraform_basics" {
  title = "Understanding Terraform basics"

  chapters = [
    resource.chapter.introduction,
  ]
}

resource "chapter" "introduction" {
  page "introduction" {
    title   = "Introduction"
    content = ""
  }
}