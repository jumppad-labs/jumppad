exec_local "install" {
  cmd = "${file_dir()}/fetch.sh"
} 
 
exec_local "run" {
  depends_on = ["exec_local.install"]

  cmd = "/tmp/consul"
  args = [
    "agent",
    "-dev",
  ]
  
  daemon = true
} 
