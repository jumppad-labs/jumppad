
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

  Scenario: Nomad Multi-Node Cluster
    Given the following shipyard variables are set
      | key               | value                 |
      | client_nodes      | 3                     |
    And I have a running blueprint
    Then the following resources should be running
      | name                      | type            |
      | cloud                     | network         |
      | server.dev                | nomad_cluster   |
      | 1.client.dev              | nomad_cluster   |
      | 2.client.dev              | nomad_cluster   |
      | 3.client.dev              | nomad_cluster   |