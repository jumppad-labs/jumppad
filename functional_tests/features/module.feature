Feature: Modules
  In order to test Modules
  I should apply a blueprint

  Scenario: Blueprint containing two modules
    Given I apply the config "./test_fixtures/modules"
    Then there should be 1 network called "onprem"
    And there should be 1 network called "cloud"
    And there should be 1 container running called "consul.container.shipyard.run"
    And there should be 1 container running called "consul.sidecar.shipyard.run"
    And there should be 1 container running called "consul-container-http.ingress.shipyard.run"
    And there should be 1 container running called "server.k3s.k8s_cluster.shipyard.run"
    And there should be 1 container running called "consul-http.ingress.shipyard.run"
    And there should be 1 container running called "vault-http.ingress.shipyard.run"
    And a call to "http://consul-http.ingress.shipyard.run:8500/v1/agent/members" should result in status 200
    And a call to "http://vault-http.ingress.shipyard.run:18200" should result in status 200
    And a call to "http://consul-http.ingress.shipyard.run:18500/v1/agent/members" should result in status 200
