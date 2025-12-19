package configuration

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"log"
	"sync"
)

var once sync.Once
var config *RestModel

func Get() (*RestModel, error) {
	if config == nil {
		log.Fatalf("Configuration not initialized.")
	}
	return config, nil

}

func Init(l logrus.FieldLogger) func(ctx context.Context) func(serviceId uuid.UUID, serviceType string) {
	return func(ctx context.Context) func(serviceId uuid.UUID, serviceType string) {
		return func(serviceId uuid.UUID, serviceType string) {
			once.Do(func() {
				c, err := requestByService(serviceId, serviceType)(l, ctx)
				if err != nil {
					log.Fatalf("Could not retrieve configuration.")
				}
				config = &c
			})
		}
	}
}
