Feature: Docker Container
  In order to test Docker containers
  something
  something

  Scenario: Single Container
    Given the config "./test_fixtures/single_container"
    When I run apply
    Then there should be 1 network called "onprem"
    And there should be 1 container running called "consul.onprem.shipyard"
    And there should be 1 container running called "consul-http.onprem.shipyard"
    And a call to "http://localhost:18500/v1/members" should result in status 200
