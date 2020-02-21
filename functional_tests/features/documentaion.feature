Feature: Docmentation
  In order to test the documentation feature
  I should create a blueprint
  and the containers should be running

  Scenario: Documentation
    Given I apply the config "./test_fixtures/docs"
    And there should be 1 container running called "docs.docs.shipyard"
    And there should be 1 container running called "terminal.docs.shipyard"
    And a call to "http://localhost:8080/" should result in status 200
