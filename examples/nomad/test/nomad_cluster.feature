
Feature: Nomad Cluster
  In order to test Nomad clusters
  I should apply a blueprint
  And test the output
  
  @single-node
  Scenario: Nomad Cluster
    Given I have a running blueprint
    Then the following resources should be running
      | name                    | type            |
      | cloud                   | network         |
      | server.dev              | nomad_cluster   |
      | consul                  | container       |
    And a HTTP call to "http://consul-http.ingress.shipyard.run:18500/v1/status/leader" should result in status 200
    And a HTTP call to "http://fake-service.ingress.shipyard.run:19090" should result in status 200
    And a HTTP call to "http://fake-service.ingress.shipyard.run:19091" should result in status 200

  @multi-node
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
      | consul                    | container       |
    And a HTTP call to "http://consul-http.ingress.shipyard.run:18500/v1/status/leader" should result in status 200
    And a HTTP call to "http://fake-service.ingress.shipyard.run:19090" should result in status 200
    And a HTTP call to "http://fake-service.ingress.shipyard.run:19091" should result in status 200
