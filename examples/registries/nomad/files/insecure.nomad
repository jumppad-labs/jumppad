job "insecure" {
  datacenters = ["dc1"]
  type        = "service"

  group "app" {
    count = 1

    network {
      port "http" {
        to     = 19092
        static = 19092
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
        LISTEN_ADDR = ":19092"
        MESSAGE     = "Registry Insecure"
      }

      config {
        image = "insecure.container.jumppad.dev:5003/mine:v0.1.0"

        ports = ["http"]
      }

      resources {
        cpu    = 500 # 500 MHz
        memory = 256 # 256MB
      }
    }
  }
}