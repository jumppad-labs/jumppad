package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupEnv(t *testing.T) (*Env, func()) {
	e, err := NewEnv("/tmp/testenv.env")
	if err != nil {
		t.Fatal(err)
	}

	return e, func() {
		e.Close()
		os.Remove("/tmp/testenv.env")
	}
}

func TestSetsEnvVar(t *testing.T) {
	envN := "tester"
	envV := "foobar"
	e, cleanup := setupEnv(t)
	defer func() {
		cleanup()
		//os.Unsetenv(envN)
	}()

	err := e.Set(envN, envV)
	assert.NoError(t, err)

	v := os.Getenv(envN)
	assert.Equal(t, envV, v)
}
