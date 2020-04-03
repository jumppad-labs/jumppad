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
	return jd.Decode(c)
}

// UnmarshalJSON is a cusom Unmarshaler to deal with
// converting the objects back into their main type
func (c *Config) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	var rawBlueprint *json.RawMessage
	json.Unmarshal(*objMap["blueprint"], &rawBlueprint)

	bp := &Blueprint{}
	err = json.Unmarshal(*rawBlueprint, &bp)
	if err == nil {
		c.Blueprint = bp
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
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeContainerIngress:
			t := ContainerIngress{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeDocs:
			t := Docs{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeExecRemote:
			t := ExecRemote{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeExecLocal:
			t := ExecLocal{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeHelm:
			t := Helm{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeIngress:
			t := Ingress{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeK8sCluster:
			t := K8sCluster{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeK8sConfig:
			t := K8sConfig{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeK8sIngress:
			t := K8sIngress{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeNetwork:
			t := Network{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeNomadCluster:
			t := NomadCluster{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeNomadJob:
			t := NomadJob{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		case TypeNomadIngress:
			t := NomadIngress{}
			err := mapstructure.Decode(mm, &t)
			if err != nil {
				return err
			}
			t.Name = mm["name"].(string)
			t.Type = ResourceType(mm["type"].(string))
			t.Status = Status(mm["status"].(string))

			if d, ok := mm["depends_on"].([]interface{}); ok {
				for _, i := range d {
					t.DependsOn = append(t.DependsOn, i.(string))
				}
			}
			c.AddResource(&t)

		}
	}

	return nil
}

// Merge config merges two config items
func (c *Config) Merge(c2 *Config) {
	for _, cc2 := range c2.Resources {
		found := false
		for i, cc := range c.Resources {
			if cc2.Info().Name == cc.Info().Name && cc2.Info().Type == cc.Info().Type {
				// Exists in the collection already
				// Replace the resource with the new one and set pending state only if it is not marked for modification.
				// If marked for modification then the user has specifically tained the resource
				status := c.Resources[i].Info().Status
				// do not update the status for resources we need to re-create or have not yet been created
				if status == Applied {
					status = PendingUpdate
				}

				c.Resources[i] = cc2
				c.Resources[i].Info().Status = status

				found = true
				break
			}
		}

		if !found {
			c.AddResource(cc2)
		}
	}

	// also merge the blueprints
	if c2.Blueprint != nil {
		c.Blueprint = c2.Blueprint
	}
}
