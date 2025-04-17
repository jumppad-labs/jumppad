package main

import (
	"os"

	"github.com/instruqt/jumppad/cmd"
)

var version = "v0.0.0"
var commit = "abc123"
var date = "0000-00-00"

func main() {
	err := cmd.Execute(version, commit, date)
	if err != nil {
		os.Exit(1)
	}
}
