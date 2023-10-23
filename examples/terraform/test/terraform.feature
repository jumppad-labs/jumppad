
Feature: Terraform provider
  In order to test the Terraform provider
  I should apply a blueprint
  And test the output
  
  Scenario: Simple example
    Given I have a running blueprint
    Then the following resources should be running
      | name                       |
      | resource.network.main      |
      | resource.container.vault   |
    And a HTTP call to "http://localhost:8200" should result in status 200
    And the following output variables should be set
      | name              | value     |
      | first             | one       |
      | second            | 2.000000  |
      | third_x           | 3.000000  |
      | third_y           | 4.000000  |
      | vault_secret      | zap       |