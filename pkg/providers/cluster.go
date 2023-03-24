package providers

import (
	"errors"
)

var (
	ErrClusterDriverNotImplemented = errors.New("driver not implemented")
	ErrClusterExists               = errors.New("cluster exists")
)
