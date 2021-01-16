package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestGenerateCreatesBundle(t *testing.T) {
	td, err := ioutil.TempDir("", "")
	defer os.RemoveAll(td)

	assert.NoError(t, err)

	fmt.Println(td)
	err = GenerateLocalBundle(td)
	assert.NoError(t, err)

	assert.FileExists(t, path.Join(td, "root.key"))
	assert.FileExists(t, path.Join(td, "root.cert"))
	assert.FileExists(t, path.Join(td, "leaf.key"))
	assert.FileExists(t, path.Join(td, "leaf.cert"))
}
