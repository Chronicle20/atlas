package configuration

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

var once sync.Once
var config *Model

func GetConfiguration() (*Model, error) {
	once.Do(func() {
		filePath := os.Getenv("CONFIG_FILE")
		file, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Unable to read the config file: %v", err)
		}
		config = &Model{}
		err = json.Unmarshal(file, config)
		if err != nil {
			log.Fatalf("Unable to parse the config file: %v", err)
		}
	})

	return config, nil
}
