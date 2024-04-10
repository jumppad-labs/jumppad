#!/bin/bash
cd "$HOME/terraform_basics" 

# Change the version of the alpine image
sed -i 's/3.16/3.17/g' docker.tf

# Apply the changes
terraform apply -auto-approve