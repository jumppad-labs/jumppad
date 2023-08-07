package resources

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// ensureAbsolute ensure that the given path is either absolute or
// if relative is converted to abasolute based on the path of the config
func ensureAbsolute(path, file string) string {
	// do not try to convert the docker sock address
	// this could happen if someone mounts the docker sock on windows
	if path == utils.GetDockerHost() {
		return path
	}

	// if the file starts with a / and we are on windows
	// we should treat this as absolute
	if runtime.GOOS == "windows" && strings.HasPrefix(path, "/") {
		return filepath.Clean(path)
	}

	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	// path is relative so make absolute using the current file path as base
	file, _ = filepath.Abs(file)

	baseDir := file
	// check if the basepath is a file return its directory
	s, _ := os.Stat(file)
	if !s.IsDir() {
		baseDir = filepath.Dir(file)
	}

	fp := filepath.Join(baseDir, path)

	return filepath.Clean(fp)
}
