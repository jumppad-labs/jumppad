Feature: Docker Continer
  In order to test Docker containers
  something
  something

  Scenario: Single Container
    Given the config "./test_fixtures/single_container"
    When I run apply
    Then there should be 1 container running
