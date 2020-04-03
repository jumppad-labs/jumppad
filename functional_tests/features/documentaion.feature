Feature: Docmentation
  In order to test the documentation feature
  I should create a blueprint
  and the containers should be running

  Scenario: Documentation
    Given I apply the config "./test_fixtures/docs"
    And there should be 1 container running called "docs.docs.shipyard.run"
    And there should be 1 container running called "terminal.docs.shipyard.run"
    And a call to "http://docs.docs.shipyard.run:8080/" should result in status 200
