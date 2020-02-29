package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var InvalidBlueprintURIError = fmt.Errorf("Inavlid blueprint URI")
var NameExceedsMaxLengthError = fmt.Errorf("Name exceeds the max length of 128 characters")
var NameContainsInvalidCharactersError = fmt.Errorf("Name contains invalid characters characters must be either a-z, A-Z, 0-9, -, _")

// ValidateName ensures that the name for a resource is within certain boundaries
// Valid characters: [a-z] [A-Z] _ - [0-9]
// Max length: 128
func ValidateName(name string) (bool, error) {
	// check the length
	if len(name) > 128 {
		return false, NameExceedsMaxLengthError
	}

	r := regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)
	ok := r.MatchString(name)
	if !ok {
		return false, NameContainsInvalidCharactersError
	}

	return true, nil
}

// FQDN generates the full qualified name for a container
func FQDN(name, typeName string) string {
	fqdn := fmt.Sprintf("%s.%s.shipyard", name, typeName)
	return fqdn
}

// FQDNVolumeName creates a full qualified volume name
func FQDNVolumeName(name string) string {
	return fmt.Sprintf("%s.volume.shipyard", name)
}

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using Kubernetes cluster
func CreateKubeConfigPath(name string) (dir, filePath string, dockerPath string) {
	dir = fmt.Sprintf("%s/.shipyard/config/%s", os.Getenv("HOME"), name)
	filePath = fmt.Sprintf("%s/kubeconfig.yaml", dir)
	dockerPath = fmt.Sprintf("%s/kubeconfig-docker.yaml", dir)

	// create the folders
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return
}

// HomeFolder returns the users homefolder this will be $HOME on windows and mac and
// USERPROFILE on windows
func HomeFolder() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}

	return os.Getenv("HOME")
}

// ShipyardHome returns the location of the shipyard
// folder, usually $HOME/.shipyard
func ShipyardHome() string {
	return fmt.Sprintf("%s/.shipyard", HomeFolder())
}

// StateDir returns the location of the shipyard
// state, usually $HOME/.shipyard/state
func StateDir() string {
	return fmt.Sprintf("%s/state", ShipyardHome())
}

// StatePath returns the full path for the state file
func StatePath() string {
	return fmt.Sprintf("%s/state.json", StateDir())
}

// IsLocalFolder tests if the given path is a localfolder and can
// exist in the current filesystem
// TODO make more robust with error messages
// to improve UX
func IsLocalFolder(path string) bool {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "./") {
		// test to see if the folder or file exists
		f, err := os.Open(path)
		if err != nil || f == nil {
			return false
		}

		return true
	}

	return false
}

// IsHCLFile tests if the given path resolves to a HCL config file
func IsHCLFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}

	if s.IsDir() {
		return false
	}

	if filepath.Ext(s.Name()) != ".hcl" {
		return false
	}

	return true
}

// GetBlueprintFolder parses a blueprint uri and returns the top level
// blueprint folder
// if the URI is not a blueprint will return an error
func GetBlueprintFolder(blueprint string) (string, error) {
	// get the folder for the blueprint
	parts := strings.Split(blueprint, "//")

	if parts == nil || len(parts) != 2 {
		fmt.Println(parts)
		return "", InvalidBlueprintURIError
	}

	return parts[1], nil
}

// GetBlueprintLocalFolder returns the full storage path
// for the given blueprint URI
func GetBlueprintLocalFolder(blueprint string) string {
	return filepath.Join(ShipyardHome(), "blueprints", blueprint)
}
