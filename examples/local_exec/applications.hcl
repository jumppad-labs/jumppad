resource "local_exec" "install" {
  command = ["${dir()}/fetch.sh"]
}

resource "local_exec" "run" {
  depends_on = ["resource.local_exec.install"]

  command = ["/tmp/consul", "agent", "-dev"]

  daemon = true
} 