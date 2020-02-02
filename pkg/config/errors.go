package config

import "errors"

// ErrorWANExists is raised when a WAN network already exists
var ErrorWANExists = errors.New("a network named 'wan' already exists")
