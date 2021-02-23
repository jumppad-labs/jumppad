Feature: Docker Container
  In order to test Shipyard local execs correctly
  I should apply a blueprint which defines a simple setup
  and test the resources are created correctly

Scenario: Two Local Execed Apps Running As Daemon
  Given I have a running blueprint
  Then a HTTP call to "http://localhost:8500/v1/status/leader" should result in status 200
