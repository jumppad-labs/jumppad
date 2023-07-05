package resources

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

func LoadState() (*hclconfig.Config, error) {
	d, err := ioutil.ReadFile(utils.StatePath())
	if err != nil {
		return hclconfig.NewConfig(), fmt.Errorf("unable to read state file: %s", err)
	}

	p := SetupHCLConfig(nil, nil, nil)
	c, err := p.UnmarshalJSON(d)
	if err != nil {
		return hclconfig.NewConfig(), fmt.Errorf("unable to unmarshal state file: %s", err)
	}

	return c, nil
}

func SaveState(c *hclconfig.Config) error {
	// save the state regardless of error
	d, err := c.ToJSON()
	if err != nil {
		return fmt.Errorf("unable to serialize config to JSON: %s", err)
	}

	err = os.MkdirAll(utils.StateDir(), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create directory for state file '%s', error: %s", utils.StateDir(), err)
	}

	err = ioutil.WriteFile(utils.StatePath(), d, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to write state file '%s', error: %s", utils.StatePath(), err)
	}

	return nil
}
