
Feature: Nomad Cluster
  In order to test Nomad clusters
  I should apply a blueprint
  And test the output

  Scenario: Nomad Cluster
    Given I apply the config "./test_fixtures/nomad"
    Then there should be 1 network called "cloud"
    And there should be 1 container running called "server.dev.nomad_cluster.shipyard.run"