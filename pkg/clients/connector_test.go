package clients

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func buildConnector(t *testing.T) string {
	// we need a shipyard binary to run for connector tests
	// build a binary
	args := []string{}

	_, filename, _, _ := runtime.Caller(0)
	dir := path.Dir(filename)

	fp := ""

	// walk backwards until we find the go.mod
	for {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return ""
		}

		for _, f := range files {
			fmt.Println("dir", dir, f.Name())
			if strings.HasSuffix(f.Name(), "go.mod") {
				fp, _ = filepath.Abs(dir)

				// found the project root
				args = []string{
					"build", "-o", "./bin/shipyardtest",
					filepath.Join(fp, "main.go"),
				}
			}
		}

		// check the parent
		dir = path.Join(dir, "../")
	}

	if len(args) == 0 {
		t.Fatal("Unable to build test binary")
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = fp

	err := cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		t.Fatal(err)
	}

	return filepath.Join(fp, "./bin/shipyardtest")
}

func TestConnectorSuite(t *testing.T) {
	//buildConnector(t)

	//c := NewConnector()

	//err := c.Start()
	//assert.NoError(t, err)
}
