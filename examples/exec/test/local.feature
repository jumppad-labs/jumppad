Feature: Locally Executed App
  In order to test jumppad local execs correctly
  I should apply a blueprint which defines a simple setup
  and test the resources are created correctly

Scenario: Two Local Execed Apps Running As Daemon
  Given I have a running blueprint
  Then a HTTP call to "http://127.0.0.1:8500/v1/status/leader" should result in status 200