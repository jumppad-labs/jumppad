package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

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

	if objMap["blueprint"] != nil {
		var rawBlueprint *json.RawMessage
		json.Unmarshal(*objMap["blueprint"], &rawBlueprint)
		bp := &Blueprint{}
		err = json.Unmarshal(*rawBlueprint, &bp)
		if err == nil {
			c.Blueprint = bp
		}
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

		var out interface{}
		switch rt := ResourceType(mm["type"].(string)); rt {
		case TypeContainerIngress:
			out = &ContainerIngress{}
		case TypeContainer:
			out = &Container{}
		case TypeDocs:
			out = &Docs{}
		case TypeExecLocal:
			out = &ExecLocal{}
		case TypeExecRemote:
			out = &ExecRemote{}
		case TypeHelm:
			out = &Helm{}
		case TypeImageCache:
			out = &ImageCache{}
		case TypeIngress:
			out = &Ingress{}
		case TypeK8sCluster:
			out = &K8sCluster{}
		case TypeK8sConfig:
			out = &K8sConfig{}
		case TypeK8sIngress:
			out = &K8sIngress{}
		case TypeModule:
			out = &Module{}
		case TypeNetwork:
			out = &Network{}
		case TypeNomadCluster:
			out = &NomadCluster{}
		case TypeNomadIngress:
			out = &NomadIngress{}
		case TypeNomadJob:
			out = &NomadJob{}
		case TypeOutput:
			out = &Output{}
		case TypeSidecar:
			out = &Sidecar{}
		case TypeTemplate:
			out = &Template{}
		case TypeCertificateCA:
			out = &CertificateCA{}
		case TypeCertificateLeaf:
			out = &CertificateLeaf{}
		case TypeCopy:
			out = &Copy{}
		case TypeVariable:
			out = &Variable{}
		default:
			return fmt.Errorf("Unable to convert to type %s, please define types in UnmarshalJSON function", rt)
		}

		err = c.decodeAndAdd(mm, out)
		if err != nil {
			return err
		}

	}

	return nil
}

func (c *Config) decodeAndAdd(in map[string]interface{}, out interface{}) error {
	dec, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:      out,
			ErrorUnused: true,
		},
	)
	if err != nil {
		return err
	}

	err = dec.Decode(in)
	if err != nil {
		return fmt.Errorf("Unable to decode into %#v, %s", out, err)
	}

	return c.AddResource(out.(Resource))
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

					if cc2.Info().Type == TypeImageCache {
						// always set Image Cache to Pending Creation to
						// force recreation to attach any new networks
						status = PendingCreation
					}
				}

				c.Resources[i] = cc2
				c.Resources[i].Info().Status = status

				// make sure the reference is the world view not the local view
				c.Resources[i].Info().Config = c

				// we need to preserve any data elements which are used to store state values
				vOld := reflect.ValueOf(cc).Elem()
				vNew := reflect.ValueOf(cc2).Elem()
				t := reflect.TypeOf(cc).Elem()

				for i := 0; i < t.NumField(); i++ {
					// Get the field tag value
					oldValue := vOld.Field(i)
					tagVal := t.Field(i).Tag.Get("state")

					// if we have a state tag copy the values from the original
					if tagVal == "true" {
						vNew.Field(i).Set(oldValue)
					}
				}

				// if image cache we should merge depends on
				// this needs to be moved to the config object
				// and should be implemented for each config type
				if cc2.Info().Type == TypeImageCache {
					if cc2.Info().DependsOn == nil {
						cc2.Info().DependsOn = []string{}
					}

					for _, dOld := range cc.Info().DependsOn {
						var found = false
						for _, dNew := range cc2.Info().DependsOn {
							if dOld == dNew {
								found = true
								break
							}
						}

						if !found {
							cc2.Info().DependsOn = append(cc2.Info().DependsOn, dOld)
						}
					}
				}

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
