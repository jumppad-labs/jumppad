package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestConfig(t *testing.T, contents string) (*Config, string, func()) {
	dir, cleanup := createTestFiles(t)
	createNamedFile(t, dir, "*.hcl", contents)

	c := New()
	err := ParseFolder(dir, c, false, "", false, []string{}, nil, "")
	assert.NoError(t, err)

	err = ParseReferences(c)
	assert.NoError(t, err)

	return c, dir, cleanup
}
