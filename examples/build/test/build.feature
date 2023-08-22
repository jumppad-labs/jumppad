Feature: Build Docker Images
  In order to test Shipyard can build images
  I should apply a blueprint which defines a simple container setup
  and test the resources are created correctly

Scenario: Build Image and Create Docker Container
  Given the following jumppad variables are set
    | key                    | value                 |
    | container_enabled      | true                  |
    | nomad_enabled          | false                 |
    | kubernetes_enabled     | false                 |
    | container_ingress_port | 19090                 |
  And I have a running blueprint
  Then the following resources should be running
    | name                                       |
    | module.container.resource.container.app  |
  And a HTTP call to "http://app.local.jumppad.dev:19090/" should result in status 200

Scenario: Build Image and Load to Nomad Cluster
  Given the following jumppad variables are set
    | key                | value                 |
    | container_enabled  | false                 |
    | nomad_enabled      | true                  |
    | kubernetes_enabled | false                 |
    | nomad_ingress_port | 19090                 |
  And I have a running blueprint
  Then the following resources should be running
    | name                      |
    | module.nomad.resource.nomad_cluster.dev  |
  And a HTTP call to "http://app.local.jumppad.dev:19090/" should result in status 200

Scenario: Build Image and Load to Kubernetes Cluster
  Given the following jumppad variables are set
    | key                     | value                 |
    | container_enabled       | true                  |
    | nomad_enabled           | false                 |
    | kubernetes_enabled      | true                  |
    | kubernetes_ingress_port | 19090                 |
  And I have a running blueprint
  Then the following resources should be running
    | name                      |
    | module.kubernetes.resource.k8s_cluster.dev  |
  And a HTTP call to "http://app.local.jumppad.dev:19090/" should result in status 200