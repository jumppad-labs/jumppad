
Feature: Custom Docker Registries
  In order to test custom docker registies with Nomad and Kubernetes
  I should apply a blueprint
  And test the output
  
  @nomad
  Scenario: Nomad Cluster
    Given the following jumppad variables are set
      | key               | value                 |
      | nomad_enabled     | true                  |
      | k8s_enabled       | false                 |
    And I have a running blueprint
    Then the following resources should be running
      | name                       |
      | resource.network.cloud     |
      | module.nomad.resource.nomad_cluster.dev |
    And a HTTP call to "http://127.0.0.1:19090" should result in status 200
    And a HTTP call to "http://127.0.0.1:19091" should result in status 200
    And a HTTP call to "http://127.0.0.1:19092" should result in status 200
  
  @k8s
  Scenario: Kubernetes Cluster
    Given the following jumppad variables are set
      | key               | value                 |
      | nomad_enabled     | false                 |
      | k8s_enabled       | true                  |
    And I have a running blueprint
    Then the following resources should be running
      | name                       |
      | resource.network.cloud     |
      | module.k8s.resource.k8s_cluster.k3s   |
    And a HTTP call to "http://127.0.0.1:29090" should result in status 200
    And a HTTP call to "http://127.0.0.1:29091" should result in status 200
    And a HTTP call to "http://127.0.0.1:29092" should result in status 200