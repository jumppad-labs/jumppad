package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var InvalidBlueprintURIError = fmt.Errorf("Inavlid blueprint URI")
var NameExceedsMaxLengthError = fmt.Errorf("Name exceeds the max length of 128 characters")
var NameContainsInvalidCharactersError = fmt.Errorf("Name contains invalid characters characters must be either a-z, A-Z, 0-9, -, _")

// ImageVolumeName is the name of the volume which stores the images for clusters
const ImageVolumeName string = "images"

// Creates the required file structure in the users Home directory
func CreateFolders() {
	os.MkdirAll(GetReleasesFolder(), os.FileMode(0755))
}

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

// ReplaceNonURIChars replaces any characters in the resrouce name which
// can not be used in a URI
func ReplaceNonURIChars(s string) (string, error) {
	reg, err := regexp.Compile(`[^a-zA-Z0-9\-\.]+`)
	if err != nil {
		return "", err
	}

	return reg.ReplaceAllString(s, "-"), nil
}

// FQDN generates the full qualified name for a container
func FQDN(name, typeName string) string {
	// ensure that the name is valid for URI schema
	cleanName, err := ReplaceNonURIChars(name)
	if err != nil {
		panic(err)
	}

	fqdn := fmt.Sprintf("%s.%s.shipyard.run", cleanName, typeName)
	return fqdn
}

// FQDNVolumeName creates a full qualified volume name
func FQDNVolumeName(name string) string {
	// ensure that the name is valid for URI schema
	cleanName, err := ReplaceNonURIChars(name)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s.volume.shipyard.run", cleanName)
}

// CreateKubeConfigPath creates the file path for the KubeConfig file when
// using Kubernetes cluster
func CreateKubeConfigPath(name string) (dir, filePath string, dockerPath string) {
	dir = fmt.Sprintf("%s/.shipyard/config/%s", HomeFolder(), name)
	filePath = fmt.Sprintf("%s/kubeconfig.yaml", dir)
	dockerPath = fmt.Sprintf("%s/kubeconfig-docker.yaml", dir)

	// create the folders
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return
}

// CreateClusterConfigPath creates the file path for the Cluster config
// which stores details such as the API server location
func CreateClusterConfigPath(name string) (dir, filePath string) {
	dir = fmt.Sprintf("%s/.shipyard/config/%s", HomeFolder(), name)
	filePath = fmt.Sprintf("%s/config.json", dir)

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

// ShipyardTemp returns a temporary folder
func ShipyardTemp() string {
	dir := filepath.Join(ShipyardHome(), "/tmp")
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}

	return dir
}

// StateDir returns the location of the shipyard
// state, usually $HOME/.shipyard/state
func StateDir() string {
	return fmt.Sprintf("%s/state", ShipyardHome())
}

// CertsDir returns the location of the certificates
// used to secure the Shipyard ingress, usually $HOME/.shipyard/certs
func CertsDir() string {
	return fmt.Sprintf("%s/certs", ShipyardHome())
}

// StatePath returns the full path for the state file
func StatePath() string {
	return fmt.Sprintf("%s/state.json", StateDir())
}

// ImageCacheLog returns the location of the image cache log
func ImageCacheLog() string {
	return fmt.Sprintf("%s/images.log", ShipyardHome())
}

// IsLocalFolder tests if the given path is a localfolder and can
// exist in the current filesystem
// TODO make more robust with error messages
// to improve UX
func IsLocalFolder(path string) bool {
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") {
		// test to see if the folder or file exists
		f, err := os.Stat(path)
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

// GetHelmLocalFolder returns the full storage path
// for the given blueprint URI
func GetHelmLocalFolder(blueprint string) string {
	return filepath.Join(ShipyardHome(), "helm_charts", blueprint)
}

// GetReleasesFolder return the path of the Shipyard releases
func GetReleasesFolder() string {
	return path.Join(ShipyardHome(), "releases")
}

// GetDataFolder creates the data directory used by the application
func GetDataFolder(p string) string {
	data := path.Join(ShipyardHome(), "data", p)
	// create the folder if it does not exist
	os.MkdirAll(data, os.ModePerm)
	return data
}

// GetDockerHost returns the location of the Docker API depending on the platform
func GetDockerHost() string {
  if dh := os.Getenv("DOCKER_HOST"); dh != "" {
    return dh
  }

	return "/var/run/docker.sock"
}

// GetConnectorPIDFile returns the connector PID file used by the connector
func GetConnectorPIDFile() string {
	return filepath.Join(ShipyardHome(), "connector.pid")
}

// GetConnectorLogFile returns the log file used by the connector
func GetConnectorLogFile() string {
	return filepath.Join(ShipyardHome(), "connector.log")
}

// GetShipyardBinaryPath returns the path to the running Shipyard binary
func GetShipyardBinaryPath() string {
	if os.Getenv("GO_ENV") == "testing" {
		_, filename, _, _ := runtime.Caller(0)
		dir := path.Dir(filename)

		// walk backwards until we find the go.mod
		for {
			files, err := ioutil.ReadDir(dir)
			if err != nil {
				return ""
			}

			for _, f := range files {
				fmt.Println("dir", dir, f.Name())
				if strings.HasSuffix(f.Name(), "go.mod") {
					fp, _ := filepath.Abs(dir)

					// found the project root
					return "go run " + filepath.Join(fp, "main.go")
				}
			}

			// check the parent
			dir = path.Join(dir, "../")
		}
	}

	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)

	return exePath
}

// GetDockerIP returns the location of the Docker Server IP address
func GetDockerIP() string {
  if dh := os.Getenv("DOCKER_HOST"); dh != "" {
    if strings.HasPrefix(dh, "tcp://") {
      u,err := url.Parse(dh)
      if err == nil {
        return strings.Split(u.Host,":")[0]
      }
    }
  }

	return "localhost"
