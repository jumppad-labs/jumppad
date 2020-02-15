package providers

import (
	"errors"
)

var (
	ErrorClusterDriverNotImplemented = errors.New("driver not implemented")
	ErrorClusterExists               = errors.New("cluster exists")
)
