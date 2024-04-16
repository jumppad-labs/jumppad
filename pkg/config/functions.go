package config

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/jumppad-labs/jumppad/pkg/utils"
)

func customHCLFuncJumppad() (string, error) {
	return utils.JumppadHome(), nil
}

// returns the docker host ip address
func customHCLFuncDockerIP() (string, error) {
	return utils.GetDockerIP(), nil
}

func customHCLFuncDockerHost() (string, error) {
	return utils.GetDockerHost(), nil
}

func customHCLFuncDataFolderWithPermissions(name string, permissions int) (string, error) {
	if permissions > 0 && permissions < 778 {
		return "", fmt.Errorf("permissions must be a three digit number less than 777")
	}

	// convert the permissions to an octal e.g. 777 to 0777
	strInt := fmt.Sprintf("0%d", permissions)
	oInt, _ := strconv.ParseInt(strInt, 8, 64)

	perms := os.FileMode(oInt)
	return utils.DataFolder(name, perms), nil
}

func customHCLFuncDataFolder(name string) (string, error) {
	perms := os.FileMode(0775)
	return utils.DataFolder(name, perms), nil
}

func customHCLFuncSystem(property string) (string, error) {
	switch property {
	case "os":
		return runtime.GOOS, nil
	case "arch":
		return runtime.GOARCH, nil
	default:
		return "", fmt.Errorf("unknown system property %s", property)
	}
}
