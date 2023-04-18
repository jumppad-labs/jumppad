Feature: Multiple Kubernetes Clusters
  In order to test Kubernetes clusters
  I should apply a blueprint
  And test the output

  Scenario: Multiple K3s Clusters With Consul
    Given I have a running blueprint
    Then the following resources should be running
      | name                                       |
      | resource.network.cloud                     |
      | module.consul_dc1.resource.k8s_cluster.dev |
      | module.consul_dc2.resource.k8s_cluster.dev |
    And a HTTP call to "http://localhost:18500/v1/agent/members" should result in status 200
    And a HTTP call to "http://localhost:18501/v1/agent/members" should result in status 200