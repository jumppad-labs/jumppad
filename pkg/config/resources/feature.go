package resources

import "github.com/jumppad-labs/hclconfig/types"

type Feature struct {
	types.ResourceBase `hcl:",remain" hclconfig:"alias=feature,omitlabel"`
}

type Test struct {
	types.ResourceBase `hcl:",remain" hclconfig:"alias=test"`
}

/*
	resource "feature" "blah" {}
	feature "blah" {}
	test "exec" "run_command" {}
*/
