 exec_local "exec_app_a" {
  cmd = "sleep"
  args = [
    "60",
  ]

  env {
    key = "CONSUL_HTTP_ADDR"
    value = "http://consul.container.shipyard.run:8500"
  }
  
  daemon = true
} 
 
 exec_local "exec_app_b" {
  cmd = "sleep"
  args = [
    "60",
  ]

  env {
    key = "CONSUL_HTTP_ADDR"
    value = "http://consul.container.shipyard.run:8500"
  }
  
  daemon = true
} 
