Feature: Multiple Kubernetes Clusters
  In order to test Kubernetes clusters
  I should apply a blueprint
  And test the output

  Scenario: Multiple K3s Clusters With Consul
    Given I have a running blueprint
    Then the following resources should be running
      | name                      | type        |
      | cloud                     | network     |
      | server.dc1                | k8s_cluster |
      | server.dc2                | k8s_cluster |
      | consul-http-dc1           | ingress     |
      | consul-http-dc2           | ingress     |
    And a HTTP call to "http://localhost:18500/v1/agent/members" should result in status 200
    And a HTTP call to "http://localhost:18501/v1/agent/members" should result in status 200