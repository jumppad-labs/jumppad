Feature: Remote Exec
  In order to test remote exec capabilities
  I should apply a blueprint
  And test the output
  
  Scenario: Remote exec
    Given I have a running blueprint
    And the following resources should be running
      | name                              |
      | resource.container.alpine         |
    When I run the script
    ```
    #!/bin/bash

    if [ ! -f $HOME/.jumppad/data/test/standalone.txt ]; then
      exit 1
    fi
    
    if [ ! -f $HOME/.jumppad/data/test/container.txt ]; then
      exit 1
    fi
    ```
    Then I expect the exit code to be 0
