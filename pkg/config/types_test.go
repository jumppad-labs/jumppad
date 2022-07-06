package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsRegisteredTypeReturnsTrue(t *testing.T) {
	val := isRegisteredType(TypeContainer)
	require.True(t, val)
}

func TestIsRegisteredTypeReturnsFalse(t *testing.T) {
	val := isRegisteredType(ResourceType("DoesNotExist"))
	require.False(t, val)
}
