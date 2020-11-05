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
  
  group "fake_service" {
    count = 1

    network {
      port  "http" { 
        to = 19090 # Dynamic port allocation
      }
    }

    restart {
      # The number of attempts to run the job within the specified interval.
      attempts = 2
      interval = "30m"
      delay = "15s"
      mode = "fail"
    }
    
    ephemeral_disk {
      size = 30
    }

    task "fake_service" {
      # The "driver" parameter specifies the task driver that should be used to
      # run the task.
      driver = "docker"
      
      logs {
        max_files     = 2
        max_file_size = 10
      }

      env {
        LISTEN_ADDR = ":19090"
        NAME = "Example2"
      }

      config {
        image = "nicholasjackson/fake-service:v0.18.1"

        ports = ["http"]
      }

      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB

      }
    }
  }
}
