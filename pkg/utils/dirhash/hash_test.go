package dirhash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratesTheCorrectHashFromADirectory(t *testing.T) {
	h, err := HashDir("./example_dir/simple", "", DefaultHash)
	require.NoError(t, err)

	require.Equal(t, "h1:1NijvqnVlOQlCYzGJAUSN5cgKgUtvUP2hWUKEw77TMc=", h)
}

func TestGeneratesTheCorrectHashFromADirectoryWithIgnore(t *testing.T) {
	h, err := HashDir("./example_dir/simple", "", DefaultHash, "**/consul_config")
	require.NoError(t, err)

	require.Equal(t, "h1:EZNhwtsE309e2xEmdMvkpxiyej1F+kwE0WnFp24YQAw=", h)
}

func TestGeneratesTheCorrectHashFromADirectoryContainingSymlinks(t *testing.T) {
	h, err := HashDir("./example_dir/symlink", "", DefaultHash)
	require.NoError(t, err)

	require.Equal(t, "h1:Bj2BThwDprZdE7mLdaWYLu2e+HY69QSUk+q3V9+DSaw=", h)

}
