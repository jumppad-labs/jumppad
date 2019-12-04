Feature: Docmentation
  In order to test the documentation feature
  something
  something

  Scenario: Documentation
    Given the config "./test_fixtures/docs"
    When I run apply
    Then there should be 1 network called "wan"
    And there should be 1 container running called "docs.wan.shipyard"
    And a call to "http://localhost:8080/" should result in status 200
