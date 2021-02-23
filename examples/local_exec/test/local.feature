Feature: Docker Container
  In order to test Shipyard local execs correctly
  I should apply a blueprint which defines a simple setup
  and test the resources are created correctly

Scenario: Two Local Execed Apps Running As Daemon
    Given I have a running blueprint
    When I run the script 
      ```
      #!/bin/bash   
      ps -a | grep sleep
      ```
    Then I expect the exit code to be 0
