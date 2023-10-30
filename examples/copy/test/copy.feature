Feature: Copying files
  In order to test copy capabilities
  I should apply a blueprint
  And check that the files have been copied

  Scenario: Copy files from multiple sources
    Given I have a running blueprint
    When I run the script
    ```
    #!/bin/bash
    if [ ! -f $HOME/.jumppad/data/copy/local/foo ]; then
      exit 1
    fi
    
    if [ ! -f $HOME/.jumppad/data/copy/http/twenty20_b4e89a76-af70-4567-b92a-9c3bbf335cb3.jpg ]; then
      exit 1
    fi

    if [ ! -f $HOME/.jumppad/data/copy/git/README.md ]; then
      exit 1
    fi

    if [ ! -f $HOME/.jumppad/data/copy/zip/nomad ]; then
      exit 1
    fi
    ```
    Then I expect the exit code to be 0