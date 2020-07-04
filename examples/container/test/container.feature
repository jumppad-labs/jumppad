Feature: Docker Container
  In order to test Shipyard creates containers correctly
  I should apply a blueprint which defines a simple container setup
  and test the resources are created correctly

  Scenario: Single Container from Local Blueprint
    Given I have a running blueprint
    Then the following resources should be running
      | name                      | type      |
      | onprem                    | network   |
      | consul                    | container |
      | consul                    | sidecar   |
      | consul-container-http     | ingress   |
    And a HTTP call to "http://consul-http.ingress.shipyard.run:8500/v1/agent/members" should result in status 200