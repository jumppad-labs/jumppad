resource "docs" "docs" {
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