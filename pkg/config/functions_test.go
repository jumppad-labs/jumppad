package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExistsFalse(t *testing.T) {
	exists, err := customHCLFuncExists("nonexistent")
	require.NoError(t, err)
	require.Equal(t, false, exists)
}

func TestExistsTrue(t *testing.T) {
	file := filepath.Join(t.TempDir(), "testdata")
	os.WriteFile(file, []byte("test"), 0644)

	exists, err := customHCLFuncExists(file)
	require.NoError(t, err)
	require.Equal(t, true, exists)
}
