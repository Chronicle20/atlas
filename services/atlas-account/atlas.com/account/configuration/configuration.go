package configuration

import (
	"sync"
)

type Registry struct {
	c *Configuration
	e error
}

type Configuration struct {
	AutomaticRegister bool   `yaml:"automaticRegister"`
	MaxPinAttempts    int    `yaml:"maxPinAttempts"`
	PinBanDuration    string `yaml:"pinBanDuration"`
	MaxPicAttempts    int    `yaml:"maxPicAttempts"`
	PicBanDuration    string `yaml:"picBanDuration"`
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
