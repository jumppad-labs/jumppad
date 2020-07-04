
Feature: Nomad Cluster
  In order to test Nomad clusters
  I should apply a blueprint
  And test the output

  Scenario: Nomad Cluster
    Given I have a running blueprint
    Then the following resources should be running
      | name                    | type            |
      | cloud                   | network         |
      | server.dev              | nomad_cluster   |