#!/bin/bash -e

# Check that the terraform apply command is executed
validate history contains "terraform apply" \
  || validate fail "'terraform apply' command was not used to apply changes"

# Is the docker alpine image present in the state?
terraform -chdir="$HOME/terraform_basics" show -json | jq -e '.values.root_module.resources | map(select(.address == "docker_image.alpine" and .values.name == "alpine:3.16")) | length == 1' > /dev/null \
  || validate fail "docker_image.alpine not found in terraform state"

# Is the docker alpine container present in the state?
terraform -chdir="$HOME/terraform_basics" show -json | jq -e '.values.root_module.resources | map(select(.address == "docker_container.alpine" and .values.name == "terraform-basics")) | length == 1' > /dev/null \
  || validate fail "docker_container.alpine not found in terraform state"

# Is the docker alpine image present?
docker image ls --format '{{json .}}' | jq -s -e '. | map(select(.Repository == "alpine" and .Tag == "3.16")) | length == 1' > /dev/null \
  || validate fail "the docker \"alpine\" image with tag \"3.16\" was not pulled"

# Is the docker alpine container running?
IMAGE_ID=$(docker image inspect alpine:3.16 | jq -r '.[0].Id' | cut -d ':' -f 2 | awk '{print substr($0,1,12)}')
docker ps --format '{{json .}}' | jq -s -e --arg image "$IMAGE_ID" '. | map(select(.Image == $image and .Names == "terraform-basics" and .State == "running")) | length == 1' > /dev/null \
  || validate fail "the docker container named \"terraform-basics\" is not running"

# If we made it this far, the solution is valid
validate success