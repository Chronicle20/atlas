package configuration

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

func loadConfiguration() (*Configuration, error) {
	yamlFile, err := ioutil.ReadFile("config.yaml")
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
