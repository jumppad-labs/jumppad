resource "http" "get" {
  method = "GET"

  url = "https://httpbin.org/get"

  headers = {
    Accept = "application/json"
  }
}

resource "http" "post" {
  method = "POST"

  url = "https://httpbin.org/post"

  payload = jsonencode({
    foo = "bar"
  })

  headers = {
    Accept = "application/json"
  }
}

output "get_body" {
  value = resource.http.get.status == 200 ? resource.http.get.body : "error"
}

output "post_body" {
  value = resource.http.post.status == 200 ? resource.http.post.body : "error"
}

