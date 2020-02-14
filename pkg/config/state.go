package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

var StateNotFoundError = fmt.Errorf("State file not found")

// ToJSON saves the config in JSON format to the specified path
// returns an error if the config can not be saved.
func (c *Config) ToJSON(path string) error {
	sd := utils.StateDir()
	sp := utils.StatePath()

	// if it does not exist create the state folder
	_, err := os.Stat(sd)
	if err != nil {
		os.MkdirAll(sd, os.ModePerm)
	}

	// if the statefile exists overwrite it
	_, err = os.Stat(sp)
	if err == nil {
		// delete the old state
		os.Remove(sp)
	}

	// serialize the state to json and write to a file
	f, err := os.Create(sp)
	if err != nil {
		return err
	}
	defer f.Close()

	ne := json.NewEncoder(f)
	return ne.Encode(c)
}

// FromJSON attempts to rehydrate the config from a JSON formatted statefile
func (c *Config) FromJSON(path string) error {
	// it is fine that the state might not exist
	f, err := os.Open(path)
	if err != nil {
		return StateNotFoundError
	}
	defer f.Close()

	jd := json.NewDecoder(f)
	jd.Decode(c)

	return nil
}

// UnmarshalJSON unmarshals the Config from a JSON string
func (c *Config) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	var rawMessagesForResources []*json.RawMessage
	err = json.Unmarshal(*objMap["resources"], &rawMessagesForResources)
	if err != nil {
		return err
	}

	for _, m := range rawMessagesForResources {
		mm := map[string]interface{}{}
		err := json.Unmarshal(*m, &mm)
		if err != nil {
			return err
		}

		t := ResourceType(mm["type"].(string))
		switch t {
		case TypeContainer:
			t := Container{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeDocs:
			t := Docs{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeExecRemote:
			t := ExecRemote{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeExecLocal:
			t := ExecLocal{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeHelm:
			t := Helm{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeIngress:
			t := Ingress{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeK8sCluster:
			t := K8sCluster{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeNetwork:
			t := Network{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		case TypeNomadCluster:
			t := NomadCluster{}
			c.addStructure(mm, &t)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) addStructure(data map[string]interface{}, dst interface{}) error {
	err := mapstructure.Decode(data, dst)
	if err != nil {
		return err
	}

	c.AddResource(dst.(Resource))
	return nil
}

// Merge config merges two config items
func (c *Config) Merge(c2 *Config) {
	for _, cc2 := range c2.Resources {
		found := false
		for _, cc := range c.Resources {
			if cc2.Info().Name == cc.Info().Name {
				// exists in the collection already set pending state
				cc.Info().Status = PendingModification
				found = true
				break
			}
		}

		if !found {
			c.AddResource(cc2)
		}
	}
}
