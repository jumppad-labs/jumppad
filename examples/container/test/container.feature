Feature: Docker Container
  In order to test Shipyard creates containers correctly
  I should apply a blueprint which defines a simple container setup
  and test the resources are created correctly

Scenario: Single Container from Local Blueprint
  Given the following environment variables are set
    | key            | value                 |
    | CONSUL_VERSION | 1.8.0                 |
    | ENVOY_VERSION  | 1.14.3                |
  And I have a running blueprint using version "v0.0.37"
  Then the following resources should be running
    | name                      | type      |
    | onprem                    | network   |
    | consul                    | container |
    | envoy                     | sidecar   |
    | consul-container-http     | ingress   |

  And the info "{.HostConfig.PortBindings['8500/'][0].HostPort}" for the running "container" running called "consul" should equal "8500"
  And the info "{.HostConfig.PortBindings['8500/'][0].HostPort}" for the running "container" running called "consul" should contain "85"
  And the info "{.HostConfig.PortBindings['8501/']}" for the running "container" running called "consul" should exist"
  And a HTTP call to "http://consul.container.shipyard.run:8500/v1/status/leader" should result in status 200
  And a HTTP call to "http://consul-http.ingress.shipyard.run:28500/v1/status/leader" should result in status 200

Scenario: Single Container from Local Blueprint with multiple runs
  Given the environment variable "CONSUL_VERSION" has a value "<consul>"
  And the environment variable "ENVOY_VERSION" has a value "<envoy>"
  And I have a running blueprint
  Then the following resources should be running
    | name                      | type      |
    | onprem                    | network   |
    | consul                    | container |
    | envoy                     | sidecar   |
    | consul-container-http     | ingress   |
  And a HTTP call to "http://consul-http.ingress.shipyard.run:8500/v1/status/leader" should result in status 200
  And the response body should contain "10.6.0.200"
  Examples:
    | consul            | envoy    |
    | 1.8.0             | 1.14.3   |
    | 1.7.3             | 1.14.3   |