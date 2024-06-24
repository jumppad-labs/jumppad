resource "docs" "docs" {
  content = [
    resource.book.terraform_basics
  ]
}

resource "book" "terraform_basics" {
  title = "Understanding Terraform basics"

  chapters = [
    resource.chapter.introduction,
    resource.chapter.installation,
  ]
}

resource "chapter" "introduction" {
  title = "Introduction"

  page "introduction" {
    content = file("./docs/index.mdx")
  }
}

resource "chapter" "installation" {
  title = "Installation"
  tasks = {
    manual_installation = resource.task.manual_installation
  }

  page "introduction" {
    content = file("./docs/terraform_basics/installation/manual_installation.mdx")
  }
}

resource "task" "manual_installation" {
  prerequisites = []

  config {
    user = "root"
  }

  condition "installed" {
    description = "Is Terraform installed"

    check {
      script = <<-EOF
        which terraform
      EOF

      failure_message = "Terraform is not installed"
    }
  }
}