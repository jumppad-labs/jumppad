package resources

import (
	"os"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

func setupState(t *testing.T, state string) {
	// set the home folder to a tmpFolder for the tests
	dir, err := ioutils.TempDir("", "")
	if err != nil {
		panic(err)
	}

	home := os.Getenv(utils.HomeEnvName())
	os.Setenv(utils.HomeEnvName(), dir)

	// write the state file
	if state != "" {
		os.MkdirAll(utils.StateDir(), os.ModePerm)
		f, err := os.Create(utils.StatePath())
		if err != nil {
			panic(err)
		}
		defer f.Close()
		_, err = f.WriteString(state)
		if err != nil {
			panic(err)
		}
	}

	t.Cleanup(func() {
		os.Setenv(utils.HomeEnvName(), home)
		os.RemoveAll(dir)
	})
}
