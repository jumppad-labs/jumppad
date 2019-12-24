package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlueprintCreatesCorrectly(t *testing.T) {
	dir, cleanup := createTestFiles(t)
	createNamedFile(t, dir, "*.yard", defaultBlueprint)
	defer cleanup()

	c := &Config{}
	err := ParseFolder(dir, c)
	assert.NoError(t, err)

	// should have created a blueprint
	bp := c.Blueprint
	assert.NotNil(t, bp)

	assert.Equal(t, "default blueprint", bp.Title)
	assert.Equal(t, "Keyser Söze", bp.Author)
	assert.Contains(t, bp.Slug, "This is")
	assert.Len(t, bp.BrowserWindows, 2)
	assert.Equal(t, "http://www.google.com", bp.BrowserWindows[0])
	assert.Len(t, bp.Environment, 2)
	assert.Equal(t, "DEBUG", bp.Environment[1].Key)
	assert.Equal(t, "true", bp.Environment[1].Value)
}

var defaultBlueprint = `
title = "default blueprint"
author = "Keyser Söze"
slug = <<EOF
	This is the slug contents
EOF

browser_windows = [
	"http://www.google.com",
	"https://www.something.com",
]

env {
	key = "KUBECONFIG"
	value = "/root/.kube/.something.yml"
}

env {
	key = "DEBUG"
	value = "true"
}
`
