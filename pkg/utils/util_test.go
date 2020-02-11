package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgIsLocalRelativeFolder(t *testing.T) {
	is := IsLocalFolder("./")

	assert.True(t, is)
}

func TestArgIsLocalAbsFolder(t *testing.T) {
	is := IsLocalFolder("/tmp")

	assert.True(t, is)
}

func TestArgIsFolderNotExists(t *testing.T) {
	is := IsLocalFolder("/dfdfdf")

	assert.False(t, is)
}

func TestArgIsNotFolder(t *testing.T) {
	is := IsLocalFolder("github.com/")

	assert.False(t, is)
}

func TestArgIsBlueprintFolder(t *testing.T) {
	dir, err := GetBlueprintFolder("github.com/org/repo//folder")

	assert.NoError(t, err)
	assert.Equal(t, "folder", dir)
}

func TestArgIsNotBlueprintFolder(t *testing.T) {
	_, err := GetBlueprintFolder("github.com/org/repo/folder")

	assert.Error(t, err)
}
