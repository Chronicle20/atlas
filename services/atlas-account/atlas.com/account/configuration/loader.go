package configuration

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
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
