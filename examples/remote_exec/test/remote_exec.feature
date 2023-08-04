Feature: Remote Exec
  In order to test remote exec capabilities
  I should apply a blueprint
  And test the output
  
  Scenario: Remote exec
    Given I have a running blueprint
    Then the following resources should be running
      | name                              |
      | resource.network.onprem           |