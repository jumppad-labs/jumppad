Feature: Certificates
  In order to test certificates are generated correctly  
  I should apply a blueprint which defines a root and a leaf certificate
  resources and test the resources are created correctly
  
@combo
Scenario: Test Root and Leaf Certificates
  Given I have a running blueprint
  When I run the script
  ```
  #!/bin/bash

  if [ ! -f $HOME/.jumppad/data/certs/root.cert ]; then
    exit 1
  fi
  
  if [ ! -f $HOME/.jumppad/data/certs/root.key ]; then
    exit 1
  fi
  
  if [ ! -f $HOME/.jumppad/data/certs/root.pub ]; then
    exit 1
  fi
  
  if [ ! -f $HOME/.jumppad/data/certs/root.ssh ]; then
    exit 1
  fi
  
  if [ ! -f $HOME/.jumppad/data/certs/nomad-leaf.cert ]; then
    exit 1
  fi
  
  if [ ! -f $HOME/.jumppad/data/certs/nomad-leaf.key ]; then
    exit 1
  fi
  
  if [ ! -f $HOME/.jumppad/data/certs/nomad-leaf.pub ]; then
    exit 1
  fi
  
  if [ ! -f $HOME/.jumppad/data/certs/nomad-leaf.ssh ]; then
    exit 1
  fi
  ```
  Then I expect the exit code to be 0