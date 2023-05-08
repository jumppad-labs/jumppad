#!/bin/bash -e

# Check that the state is empty
terraform show -json | jq -e '.Values == null' > /dev/null \
  || validate fail "the terraform state is not empty"

# Check that the docker container is removed
docker ps --format '{{json .}}' | jq -s -e '. | map(select(.Names == "terraform-basics")) | length == 0' > /dev/null \
  || validate fail "the docker container named 'terraform-basics' is still running"

# Check that the docker image is removed
docker image ls --format '{{json .}}' | jq -s -e '. | map(select(.Repository == "alpine" and .Tag == "3.17")) | length == 0' > /dev/null \
  || validate fail "the docker 'alpine' image with tag '3.17' was not removed"

# If we made it this far, the solution is valid
validate success