Feature: Multiple Blueprints
  In order to test Shipyard can create multiple blueprints in a test
  I should apply a blueprint which defines a simple container setup
  and test the resources are created correctly
  
@combo
Scenario: Test multiple blueprints
  Given the following environment variables are set
    | key            | value                 |
    | CONSUL_VERSION | 1.10.6                |
    | ENVOY_VERSION  | 1.18.4                |
  And I have a running blueprint at path "../docs"
  And I have a running blueprint
  Then the following resources should be running
    | name                                  |
    | resource.network.onprem               |
    | resource.docs.docs                    |
    | resource.container.consul             |
    | resource.sidecar.envoy                |