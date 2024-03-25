package jumppad

import "github.com/spf13/viper"

func GetDefaultRegistry() string {
	return viper.GetString("default_registry")
}

func GetRegistryCredentials() map[string]string {
	registryCredentials := map[string]string{}
	for _, registries := range viper.Get("credentials").([]map[string]interface{}) {
		for r, v := range registries {
			c := v.([]map[string]interface{})[0]
			if c["token"] != nil {
				registryCredentials[r] = c["token"].(string)
			}
		}
	}

	return registryCredentials
}
