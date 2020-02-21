Feature: Docker Container
  In order to test Docker containers
  I should apply a blueprint

  Scenario: Single Container
    Given I apply the config "./test_fixtures/single_container"
    Then there should be 1 network called "onprem"
    And there should be 1 container running called "consul.container.shipyard"
    And there should be 1 container running called "consul-http.ingress.shipyard"
    And a call to "http://localhost:18500/v1/members" should result in status 200
