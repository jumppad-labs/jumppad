Feature: Docker Container
  In order to test Shipyard can build containers
  I should apply a blueprint which defines a simple container setup
  and test the resources are created correctly

Scenario: Single Container from Local Blueprint
  Given I have a running blueprint
  Then the following resources should be running
    | name                      | type          |
    | build                     | container     |
  And the info "{.NetworkSettings.Ports['9090/tcp'][0].HostPort}" for the running "container" called "build" should equal "9090"
  And a HTTP call to "http://build.container.shipyard.run:9090/" should result in status 200