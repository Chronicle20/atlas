package configuration

import (
	"gopkg.in/yaml.v2"
	"os"
)

func loadConfiguration() (*Configuration, error) {
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	con := &Configuration{}
	err = yaml.Unmarshal(yamlFile, con)
	if err != nil {
		return nil, err
	}
	return con, nil
}
