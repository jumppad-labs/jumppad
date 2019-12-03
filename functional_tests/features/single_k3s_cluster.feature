Feature: Kubernetes Cluster
  In order to test Kubernetes clusters
  something
  something

  Scenario: K3s Cluster
    Given the config "./test_fixtures/single_k3s_cluster"
    When I run apply
    Then there should be 1 network called "cloud"
    And there should be 1 container running called "server.k3s.cloud.shipyard"