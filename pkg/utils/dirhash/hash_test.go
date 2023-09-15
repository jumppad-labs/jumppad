package dirhash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratesTheCorrectHashFromADirectory(t *testing.T) {
	h, err := HashDir("./example_dir/simple", "", DefaultHash)
	require.NoError(t, err)

	require.Equal(t, "h1:UuBnfJ+yMhiUs/lxAbzwgaPHLNFXwbJLxEbzZevtxEc=", h)
}

func TestGeneratesTheCorrectHashFromADirectoryWithIgnore(t *testing.T) {
	h, err := HashDir("./example_dir/simple", "", DefaultHash, "**/consul_config")
	require.NoError(t, err)

	require.Equal(t, "h1:o3BPjmvmay4mdDA7rkU4Le34uvkRHuCPlW5MJ9p4T2s=", h)
}

func TestGeneratesTheCorrectHashFromADirectoryContainingSymlinks(t *testing.T) {
	h, err := HashDir("./example_dir/symlink", "", DefaultHash)
	require.NoError(t, err)

	require.Equal(t, "h1:rVNDC8+SpMX32ggfqjNmud8EFVMphVqVLs0x+LcWLTA=", h)

}
