Feature: Docker Container
  In order to test jumppad creates containers correctly using variables
  I should apply a blueprint which defines a simple container setup
  and test the resources are created correctly

Scenario: Single Container with jumppad Variables
  Given the following environment variables are set
    | key            | value                 |
    | BAH            | bah                   |
  And the following jumppad variables are set
    | key            | value                 |
    | something      | set by test           |
  And I have a running blueprint
  Then the following resources should be running
    | name                                       |
    | resource.network.consul                    |
    | resource.container.consul                  |
    | resource.sidecar.envoy                     |
  And the info "{.Config.Env}" for the running container "resource.container.consul" should contain "something=set by test"
  And the info "{.Config.Env}" for the running container "resource.container.consul" should contain "foo=bah"