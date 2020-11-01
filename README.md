# Shipyard


![](https://github.com/shipyard-run/shipyard/workflows/Build/badge.svg)  
![](https://github.com/shipyard-run/shipyard/workflows/Release/badge.svg)  
  
[![codecov](https://codecov.io/gh/shipyard-run/shipyard/branch/master/graph/badge.svg)](https://codecov.io/gh/shipyard-run/shipyard)

![](./shipyard_horizontal.png)

Shipyard is a tool for building modern cloud native development environments. Using the Shipyard configuration language you can create Docker containers, Nomad/Kubernetes clusters and more. Shipyard understands terraform

## Community
Join our community on Discord: [https://discord.gg/ZuEFPJU69D](https://discord.gg/ZuEFPJU69D)

## Questions
## Is Shipyard like Terraform?
Kind of but more about local environments rather than infrastructure

Docker Compose is one of our favourite tools but we found it does not manage dependencies particulary well. Compose also works on a really low level of abstraction. Shipyard addresses these missing features.
## Why not use Docker Compose?

## Is Shipyard just for Docker?
No, Shipyard is designed to work with Docker, Podman, Raw binaries, etc. At present we only have a Driver for Docker containers but others are on our Roadmap.

## I have a huge environment how can Shipyard help?
Shipyard v0.2.0 will ship with a remote connection capability, it will allow you to connect a Shipyard stack running locally to a remote cluster.

## Can I use Shipyard for anything other than Dev environments?
Yes, Shipyard can be used to create interactive documentation for your applications and redistributable demo environments to show off your tool or product.

## Example Shipyard Config
The following snippets are examples of things you can build with Shipyard, for more detailed examples please see the Blueprints repo [https://github.com/shipyard-run/blueprints](https://github.com/shipyard-run/blueprints)

## Kubernetes Cluster

```
k8s_cluster "k3s" {
  driver  = "k3s" // default
  version = "v1.0.0"

  nodes = 1 // default

  network {
    name = "network.cloud"
  }
}

helm "consul" {
  cluster = "k8s_cluster.k3s"
  chart = "./helm/consul-helm-0.16.2"
  values = "./helm/consul-values.yaml"

  health_check {
    timeout = "60s"
    pods = ["release=consul"]
  }
}

k8s_ingress "consul-http" {
  cluster = "k8s_cluster.k3s"
  service  = "consul-consul-server"

  network {
    name = "network.cloud"
  }

  port {
    local  = 8500
    remote = 8500
    host   = 18500
  }
}
```

## Nomad Cluster

```
nomad_cluster "dev" {
  version = "v0.10.2"

  nodes = 1 // default

  network {
    name = "network.cloud"
  }
}

nomad_job "redis" {
  cluster = "nomad_cluster.dev"

  paths = ["./app_config/example2.nomad"]
  health_check {
    timeout = "60s"
    nomad_jobs = ["example_2"]
  }
}

nomad_ingress "nomad-http" {
  cluster  = "nomad_cluster.dev"
  job = ""
  group = ""
  task = ""

  port {
    local  = 4646
    remote = 4646
    host   = 14646
    open_in_browser = "/"
  }

  network  {
    name = "network.cloud"
  }
}
```

## Docker Container

```
container "consul" {
  image   {
    name = "consul:1.6.1"
  }

  command = ["consul", "agent", "-config-file=/config/consul.hcl"]

  volume {
    source      = "./consul_config"
    destination = "/config"
  }

  network {
    name = "network.onprem"
    ip_address = "10.5.0.200" // optional
  }

  resources {
    # Max CPU to consume, 1024 is one core, default unlimited
    cpu = 2048
    # Pin container to specified CPU cores, default all cores
    cpu_pin = [1,2]
    # max memory in MB to consume, default unlimited
    memory = 1024
  }

  env {
    key ="abc"
    value = "123"
  }

  env {
    key ="SHIPYARD_FOLDER"
    value = "${shipyard()}"
  }

  env {
    key ="HOME_FOLDER"
    value = "${home()}"
  }
}
```

## Contributing

We love contributions to the project, to contribute, first ensure that there is an issue and that it has been acknowledged by one of the maintainers of the project. Ensuring an issue exists and has been acknowledged ensures that the work you are about to submit will not be rejected due to specifications or duplicate work.
Once an issue exists, you can modify the code and raise a PR against this repo. We are working on increasing code coverage, please ensure that your work has at least 80% test coverage before submitting.


## Testing:

The project has two types of test, pure code Unit tests and, Functional tests which apply real blueprints to a locally running Docker engine and test output.

### Unit tests:

To run the unit tests you can use the make recipe `make test_unit` this runs the `go test` and excludes the functional tests.

```shell
shipyard on  master via 🐹 v1.13.5 on 🐳 v19.03.5 ()
➜ make test_unit
go test -v -race github.com/shipyard-run/shipyard github.com/shipyard-run/shipyard/cmd github.com/shipyard-run/shipyard/pkg/clients github.com/shipyard-run/shipyard/pkg/clients/mocks github.com/shipyard-run/shipyard/pkg/config github.com/shipyard-run/shipyard/pkg/providers github.com/shipyard-run/shipyard/pkg/shipyard github.com/shipyard-run/shipyard/pkg/utils
testing: warning: no tests to run
PASS
ok      github.com/shipyard-run/shipyard        (cached) [no tests to run]
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

### Functional tests:

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
    And there should be 1 container running called "docs.docs.shipyard.run" # main_test.go:115 -> thereShouldBeContainerRunningCalled
    And a call to "http://localhost:8080/" should result in status 200 #

# ...

3 scenarios (3 passed)
16 steps (16 passed)
3m6.79622s
testing: warning: no tests to run
PASS
```

## Creating a release:

To create a release tag a commit `git tag <semver>` and push this to GitHub `git push origin <semver>` GitHub actions will build and create the release.
