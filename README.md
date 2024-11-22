# jumppad

![build](https://github.com/jumppad-labs/jumppad/actions/workflows/build_and_deploy.yaml/badge.svg)  

![release](https://img.shields.io/github/v/release/jumppad-labs/jumppad.svg?style=flat)  

[![codecov](https://codecov.io/gh/jumppad-labs/jumppad/branch/master/graph/badge.svg)](https://codecov.io/gh/jumppad-labs/jumppad)

Jumppad is a tool for building modern cloud native development environments. Using the Jumppad configuration language you can create OCI containers, Nomad/Kubernetes clusters and more.

## Community

Join our community on Discord: [https://discord.gg/ZuEFPJU69D](https://discord.gg/ZuEFPJU69D)

## Questions

### Is Jumppad like Terraform?

Kind of, but more about local environments rather than infrastructure

### Why not use Docker Compose?

Docker Compose is one of our favourite tools but we found it does not manage dependencies particulary well. Compose also works on a really low level of abstraction. Jumppad addresses these missing features.

### Is Jumppad just for Docker?

No, Jumppad is designed to work with Docker, Podman, Raw binaries, etc. At present we only have a Driver for Docker and Podman, but others are on our Roadmap.

### Can I use Jumppad for anything other than Dev environments?

Yes, Jumppad can be used to create interactive documentation for your applications and redistributable demo environments to show off your tool or product.

## Example Jumppad Config

The following snippets are examples of things you can build with Jumppad, for more detailed examples please see the Blueprints repo [https://github.com/shipyard-run/blueprints](https://github.com/shipyard-run/blueprints)

### Kubernetes Cluster

```hcl
resource "network" "cloud" {
  subnet = "10.5.0.0/24"
}

resource "k8s_cluster" "k3s" {
  driver = "k3s" // default

  nodes = 1 // default

  network {
    id = resource.network.cloud.id
  }

  copy_image {
    name = "ghcr.io/jumppad-labs/connector:v0.4.0"
  }
}

resource "k8s_config" "fake_service" {
  cluster = resource.k8s_cluster.k3s

  paths = ["./fake_service.yaml"]

  health_check {
    timeout = "240s"
    pods    = ["app.kubernetes.io/name=fake-service"]
  }
}

resource "helm" "vault" {
  cluster = resource.k8s_cluster.k3s

  repository {
    name = "hashicorp"
    url  = "https://helm.releases.hashicorp.com"
  }

  chart   = "hashicorp/vault"
  version = "v0.18.0"

  values = "./helm/vault-values.yaml"

  health_check {
    timeout = "240s"
    pods    = ["app.kubernetes.io/name=vault"]
  }
}

resource "ingress" "vault_http" {
  port = 18200

  target {
    resource = resource.k8s_cluster.k3s
    port = 8200

    config = {
      service   = "vault"
      namespace = "default"
    }
  }
}

resource "ingress" "fake_service" {
  port = 19090

  target {
    resource = resource.k8s_cluster.k3s
    port = 9090

    config = {
      service   = "fake-service"
      namespace = "default"
    }
  }
}

output "VAULT_ADDR" {
  value = resource.ingress.vault_http.address
}

output "KUBECONFIG" {
  value = resource.k8s_cluster.k3s.kubeconfig
}

```

### Nomad Cluster

```hcl
resource "network" "cloud" {
  subnet = "10.10.0.0/16"
}

resource "nomad_cluster" "dev" {
  client_nodes=3

  network {
    id = resource.network.cloud.id
  }
}

resource "nomad_job" "example_1" {
  cluster = resource.nomad_cluster.dev

  paths = ["./app_config/example1.nomad"]

  health_check {
    timeout    = "60s"
    nomad_jobs = ["example_1"]
  }
}

resource "ingress" "fake_service_1" {
  port = 19090

  target {
    resource   = resource.nomad_cluster.dev
    named_port = "http"

    config = {
      job   = "example_1"
      group = "fake_service"
      task  = "fake_service"
    }
  }
}
```

### Docker Container

```hcl
resource "container" "unique_name" {
    depends_on = ["resource.container.another"]

    network {
        id         = resource.network.cloud.id
        ip_address = "10.16.0.200"
        aliases    = ["my_unique_name_ip_address"]
    }

    image {
        name     = "consul:1.6.1"
        username = "repo_username"
        password = "repo_password"
    }

    command = [
        "consul",
        "agent"
    ]

    environment = {
        CONSUL_HTTP_ADDR = "http://localhost:8500"
    }

    volume {
        source      = "./config"
        destination = "/config"
    }

    port {
        local  = 8500
        remote = 8500
        host   = 18500
    }
    
    port_range {
        range       = "9000-9002"
        enable_host = true
    }

    privileged = false
}
```

## Podman support

Podman support is experimental and at present many features such as Kubernetes clusters do not work with rootless podman and require root access.

### Enable the podman socket

Jumppad uses podman's API, which is compatible with the Docker Enginer API. To enable this, you need to run the podman socket as a group that your user has access to. The following example uses the `docker` group:

```shell
sudo sed '/^SocketMode=.*/a SocketGroup=docker' -i /lib/systemd/system/podman.socket
```

Then emable the podman socket service

```shell
sudo systemctl daemon-reload

sudo systemctl enable podman.socket
sudo systemctl enable podman.service
sudo systemctl start podman.socket
sudo systemctl start podman.service
```

For sockets to be writable they also require execute permission on the parent folder

```shell
sudo chmod +x /run/podman
```

Point your DOCKER_HOST environment variable at the socket

```shell
export DOCKER_HOST=unix:///run/podman/podman.sock
```

If you have the Docker CLI installed you should be able to contact the podman daemon using the standard Docker commands

```shell
docker ps
```

### Default network

```shell
➜ sudo podman network ls
NETWORK ID    NAME    VERSION  PLUGINS
2f259bab93aa  podman  0.4.0    bridge,portmap,firewall,tuning
```

### Registries

Image pull silently fails if there are no registries defined in podmans /etc/containers/registries.conf

```shell
echo -e "[registries.search]\nregistries = ['docker.io']" | sudo tee /etc/containers/registries.conf
```

### DNS

Multiple network cause DNS resolution problems
https://github.com/containers/podman/issues/8399

Enabling name resolution

#### Install dnsmasq

First ensure systemd is not using port 53
https://www.linuxuprising.com/2020/07/ubuntu-how-to-free-up-port-53-used-by.html

Install dnsmasq

```shell
sudo apt install dnsmasq
```

Configure dnsmasq for external name server resolution or external network connections will not work

![Install the podman dns plugin](https://github.com/containers/dnsname/blob/main/README_PODMAN.md).

## Contributing

We love contributions to the project, to contribute, first ensure that there is an issue and that it has been acknowledged by one of the maintainers of the project. Ensuring an issue exists and has been acknowledged ensures that the work you are about to submit will not be rejected due to specifications or duplicate work.
Once an issue exists, you can modify the code and raise a PR against this repo. We are working on increasing code coverage, please ensure that your work has at least 80% test coverage before submitting.

## Testing

The project has two types of tests, pure code Unit tests and Functional tests, which apply real blueprints to a locally-running Docker engine and test output.

### Unit tests

To run the unit tests you can use the make recipe `make test_unit` this runs the `go test` and excludes the functional tests.

```shell
jumppad on  master via 🐹 v1.13.5 on 🐳 v19.03.5 ()
➜ make test_unit
go test -v -race github.com/jumppad-labs/jumppad github.com/jumppad-labs/jumppad/cmd github.com/jumppad-labs/jumppad/pkg/clients github.com/jumppad-labs/jumppad/pkg/clients/mocks github.com/jumppad-labs/jumppad/pkg/config github.com/jumppad-labs/jumppad/pkg/providers github.com/jumppad-labs/jumppad/pkg/jumppad github.com/jumppad-labs/jumppad/pkg/utils
testing: warning: no tests to run
PASS
ok      github.com/jumppad-labs/jumppad        (cached) [no tests to run]
=== RUN   TestSetsEnvVar
--- PASS: TestSetsEnvVar (0.00s)
=== RUN   TestArgIsLocalRelativeFolder
--- PASS: TestArgIsLocalRelativeFolder (0.00s)
=== RUN   TestArgIsLocalAbsFolder
--- PASS: TestArgIsLocalAbsFolder (0.00s)
=== RUN   TestArgIsFolderNotExists
--- PASS: TestArgIsFolderNotExists (0.00s)
=== RUN   TestArgIsNotFolder
--- PASS: TestArgIsNotFolder (0.00s)
=== RUN   TestArgIsBlueprintFolder
--- PASS: TestArgIsBlueprintFolder (0.00s)
=== RUN   TestArgIsNotBlueprintFolder
```

### Functional tests

To run the functional tests ensure that Docker is running in your environment then run `make test_functional`. functional tests are executed with GoDog cucumber test runner for Go. Note: These tests execute real blueprints and can a few minutes to run.

```shell
➜ make test_functional
cd ./functional_tests && go test -v ./...
Feature: Docmentation
  In order to test the documentation feature
  something
  something

  Scenario: Documentation                                              # features/documentaion.feature:6
    Given the config "./test_fixtures/docs"                            # main_test.go:77 -> theConfig
2020-02-08T17:03:25.269Z [INFO]  Creating Network: ref=wan
2020-02-08T17:03:40.312Z [INFO]  Creating Documentation: ref=docs
2020-02-08T17:03:40.312Z [INFO]  Creating Container: ref=docs
2020-02-08T17:03:40.490Z [DEBUG] Attaching container to network: ref=docs network=wan
2020-02-08T17:03:41.187Z [INFO]  Creating Container: ref=terminal
2020-02-08T17:03:41.271Z [DEBUG] Attaching container to network: ref=terminal network=wan
    When I run apply                                                   # main_test.go:111 -> iRunApply
    Then there should be 1 network called "wan"                        # main_test.go:149 -> thereShouldBe1NetworkCalled
    And there should be 1 container running called "docs.docs.jumppad.dev" # main_test.go:115 -> thereShouldBeContainerRunningCalled
    And a call to "http://localhost:8080/" should result in status 200 #

# ...

3 scenarios (3 passed)
16 steps (16 passed)
3m6.79621s
testing: warning: no tests to run
PASS
```

## Creating a release

To create a release tag a commit `git tag <semver>` and push this to GitHub `git push origin <semver>` GitHub actions will build and create the release.
