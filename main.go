package main

import (
	"github.com/shipyard-run/shipyard/cmd"
)

var version = "dev"

func main() {
	cmd.Execute(version)
}