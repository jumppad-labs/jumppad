job "auth" {
  datacenters = ["dc1"]
  type        = "service"

  group "app" {
    count = 1

    network {
      port "http" {
        to     = 19091
        static = 19091
      }
    }

    ephemeral_disk {
      size = 30
    }

    task "app" {
      # The "driver" parameter specifies the task driver that should be used to
      # run the task.
      driver = "docker"

      logs {
        max_files     = 2
        max_file_size = 10
      }

      env {
        LISTEN_ADDR = ":19091"
        MESSAGE     = "Registry With Auth"
      }

      config {
        image = "auth-registry.demo.gs/mine:v0.1.0"

        ports = ["http"]
      }

      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB
      }
    }
  }
}