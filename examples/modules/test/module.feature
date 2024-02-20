Feature: Modules
  In order to test Modules
  I should apply a blueprint

  Scenario: Blueprint containing two modules
    Given the following environment variables are set
      | key            | value                 |
      | CONSUL_VERSION | 1.8.0                 |
      | ENVOY_VERSION  | 1.14.3                |
    And I have a running blueprint
    Then the following resources should be running
      | name                                    |
      | resource.network.cloud                  |
      | resource.network.onprem                 |
      | resource.container.consul               |
      | resource.sidecar.envoy                  |
      | resource.k8s_cluster.k3s                |
      | resource.docs.docs                      |
    And a HTTP call to "http://consul.container.shipyard.run:8500/v1/agent/members" should result in status 200
    And a HTTP call to "http://consul-http.ingress.shipyard.run:18500/v1/agent/members" should result in status 200
