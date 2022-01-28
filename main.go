package main

import (
	"github.com/shipyard-run/shipyard/cmd"
)

var version = "v0.0.0"
var commit = "abc123"
var date = "0000-00-00"

func main() {
	cmd.Execute(version, commit, date)
}
