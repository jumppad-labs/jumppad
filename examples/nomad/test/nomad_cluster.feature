
Feature: Nomad Cluster
  In order to test Nomad clusters
  I should apply a blueprint
  And test the output
  
  @single-node
  Scenario: Nomad Cluster
    Given I have a running blueprint
    Then the following resources should be running
      | name                       |
      | resource.network.cloud     |
      | resource.nomad_cluster.dev |
      | resource.container.consul                   |
    And a HTTP call to "http://localhost:18500/v1/status/leader" should result in status 200
    And a HTTP call to "http://localhost:19091" should result in status 200
  
  @datacenter
  Scenario: Nomad Cluster
    Given the following jumppad variables are set
      | key               | value                 |
      | datacenter        | dc2                   |
    And I have a running blueprint
    Then the following resources should be running
      | name                       |
      | resource.network.cloud     |
      | resource.nomad_cluster.dev |
      | resource.container.consul                   |
    And a HTTP call to "http://localhost:18500/v1/status/leader" should result in status 200
    And a HTTP call to "http://localhost:19091" should result in status 200

  @multi-node
  Scenario: Nomad Multi-Node Cluster
    Given the following jumppad variables are set
      | key               | value                 |
      | client_nodes      | 3                     |
    And I have a running blueprint
    Then the following resources should be running
      | name                                        |
      | resource.network.cloud                      |
      | resource.nomad_cluster.dev                  |
      | resource.container.consul                   |
    And a HTTP call to "http://localhost:18500/v1/status/leader" should result in status 200
    And a HTTP call to "http://localhost:19091" should result in status 200
