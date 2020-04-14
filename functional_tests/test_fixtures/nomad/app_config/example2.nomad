job "example_2" {
  datacenters = ["dc1"]
  type = "service"

  update {
    max_parallel = 1
    min_healthy_time = "10s"
    healthy_deadline = "3m"
    progress_deadline = "10m"
    auto_revert = false
    canary = 0
  }
  
  migrate {
    max_parallel = 1
    health_check = "checks"
    min_healthy_time = "10s"
    healthy_deadline = "5m"
  }
  
  group "consul" {
    count = 1

    restart {
      # The number of attempts to run the job within the specified interval.
      attempts = 2
      interval = "30m"
      delay = "15s"
      mode = "fail"
    }
    
    ephemeral_disk {
      size = 200
    }

    task "consul" {
      # The "driver" parameter specifies the task driver that should be used to
      # run the task.
      driver = "docker"

      config {
        image = "consul:1.7.1"

        port_map {
          http = 8500
        }
      }

      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB

        network {
          mbits = 10
          port  "http"  {}
        }
      }
    }
  }
}
