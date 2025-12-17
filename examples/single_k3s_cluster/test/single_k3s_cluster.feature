Feature: Kubernetes Cluster
  In order to test Kubernetes clusters
  I should apply a blueprint
  And test the output

  Scenario: K3s Cluster
    Given I have a running blueprint
    Then the following resources should be running
      | name                                    |
      | resource.network.cloud                  |
      | resource.k8s_cluster.k3s                |
    And a HTTP call to "http://127.0.0.1:18500/v1/agent/members" should result in status 200
    And a HTTP call to "http://127.0.0.1:18200" should result in status 200
    And a TCP connection to "127.0.0.1:8300" should open
