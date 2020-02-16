package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func setupConfigTests(t *testing.T) (*Config, func()) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	home := os.Getenv("HOME")
	os.Setenv("HOME", dir)

	// create a config with all resource types
	c := New()
	c.AddResource(NewContainer("config"))
	c.AddResource(NewDocs("config"))
	c.AddResource(NewExecLocal("config"))
	c.AddResource(NewExecRemote("config"))
	c.AddResource(NewHelm("config"))
	c.AddResource(NewIngress("config"))
	c.AddResource(NewK8sCluster("config"))
	c.AddResource(NewNetwork("config"))
	c.AddResource(NewNomadCluster("config"))

	return c, func() {
		os.Setenv("HOME", home)
		os.RemoveAll(dir)
	}
}

func TestConfigSerializesToJSON(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	statePath := utils.StatePath()
	err := c.ToJSON(statePath)

	assert.NoError(t, err)

	// check the file
	c2 := New()
	d, err := ioutil.ReadFile(statePath)
	assert.NoError(t, err)

	fmt.Println(string(d))
	err = json.Unmarshal(d, c2)
	assert.NoError(t, err)
	assert.Len(t, c2.Resources, c.ResourceCount())
}

func TestConfigDeSerializesFromJSON(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	statePath := utils.StatePath()
	err := c.ToJSON(statePath)

	c = New()
	err = c.FromJSON(statePath)
	assert.NoError(t, err)

	assert.Len(t, c.Resources, 9)
	assert.Equal(t, ResourceType("container"), c.Resources[0].Info().Type)
	assert.Equal(t, "config", c.Resources[0].Info().Name)
}

func TestConfigMergesAddingItems(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c2 := New()
	c2.AddResource(NewContainer("test"))

	c.Merge(c2)

	assert.Len(t, c.Resources, 10)
}

func TestConfigMergesWithExistingItemSetsPendingUpdateWhenApplied(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c.Resources[0].Info().Status = Applied

	c2 := New()
	c2.AddResource(NewContainer("config"))

	c.Merge(c2)

	assert.Len(t, c.Resources, 9)
	assert.Equal(t, c.Resources[0].Info().Status, PendingUpdate)
}

func TestConfigMergesWithExistingItemDoesNOTSetsPendingUpdateWhenOtherStatus(t *testing.T) {
	c, cleanup := setupConfigTests(t)
	defer cleanup()

	c.Resources[0].Info().Status = PendingCreation

	c2 := New()
	c2.AddResource(NewContainer("config"))

	c.Merge(c2)

	assert.Len(t, c.Resources, 9)
	assert.Equal(t, c.Resources[0].Info().Status, PendingCreation)
}
