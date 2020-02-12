package utils

import (
	"testing"

	"github.com/gosuri/uitable/util/strutil"
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

func TestValidatesNameCorrectly(t *testing.T) {
	ok, err := ValidateName("abc-sdf")
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestValidatesNameAndReturnsErrorWhenInvalid(t *testing.T) {
	ok, err := ValidateName("*$-abcd")
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestValidatesNameAndReturnsErrorWhenTooLong(t *testing.T) {
	dn := strutil.PadLeft("a", 128, 'a'))

	ok, err := ValidateName(dn)

	assert.Error(t, err)
	assert.False(t, ok)
}
