package configuration

import (
	"sync"
)

type Registry struct {
	c *Configuration
	e error
}

type Configuration struct {
	AutomaticRegister bool `yaml:"automaticRegister"`
}

var configurationRegistryOnce sync.Once
var configurationRegistry *Registry

func Get() (*Configuration, error) {
	configurationRegistryOnce.Do(func() {
		configurationRegistry = &Registry{}
		c, err := loadConfiguration()
		configurationRegistry.c = c
		configurationRegistry.e = err
	})
	return configurationRegistry.c, configurationRegistry.e
}
