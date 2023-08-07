package resources

import "github.com/jumppad-labs/hclconfig/types"

type Feature struct {
	types.ResourceMetadata `hcl:",remain" hclconfig:"alias=feature,omitlabel"`
}

type Test struct {
	types.ResourceMetadata `hcl:",remain" hclconfig:"alias=test"`
}

/*
	resource "feature" "blah" {}
	feature "blah" {}
	test "exec" "run_command" {}
*/
