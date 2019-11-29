package config

import "errors"

var ErrorWANExists = errors.New("a network named 'wan' already exists")
