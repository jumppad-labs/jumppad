package config

import (
	"os"
	"testing"
)

// createsTestFiles creates a temporary directory and
// stores temp files into it
// returns directory containing files
// cleanup function
// usage:
// d, cleanup := createTestFiles(t, `cluster "abc" {}`, `docs "bcdf" {}`)
// defer cleanup()
func CreateTestFiles(t *testing.T, contents ...string) string {
	dir := createTempDirectory(t)

	for _, x := range contents {
		createNamedFile(t, dir, "*.hcl", x)
	}

	t.Cleanup(func() {
		removeTestFiles(t, dir)
	})

	return dir
}

// createTestFile creates a hcl file from the given contents
func CreateTestFile(t *testing.T, contents string) string {
	dir := createTempDirectory(t)

	t.Cleanup(func() {
		removeTestFiles(t, dir)
	})

	return createNamedFile(t, dir, "*.hcl", contents)
}

// create a temporary directory
func createTempDirectory(t *testing.T) string {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %s", err)
	}

	return dir
}

func createNamedFile(t *testing.T, dir, name, contents string) string {
	f, err := os.CreateTemp(dir, name)
	if err != nil {
		t.Fatalf("Error creating temp file %s", err)
	}
	defer f.Close()

	if _, err := f.WriteString(contents); err != nil {
		t.Fatalf("Error writing temp file contents: %s", err)
	}

	return f.Name()
}

// remove test files cleans up any temporary files created
// with createTestFile
func removeTestFiles(t *testing.T, dir string) {
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("Unable to remove temporary files %s", err)
	}
}
