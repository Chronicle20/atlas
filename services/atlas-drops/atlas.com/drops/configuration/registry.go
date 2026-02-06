package configuration

import (
	"context"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var once sync.Once
var serviceConfig *RestModel

func GetServiceConfig() (*RestModel, error) {
	if serviceConfig == nil {
		log.Fatalf("Configuration not initialized.")
	}
	return serviceConfig, nil
}

func Init(l logrus.FieldLogger) func(ctx context.Context) func(serviceId uuid.UUID) {
	return func(ctx context.Context) func(serviceId uuid.UUID) {
		return func(serviceId uuid.UUID) {
			once.Do(func() {
				c, err := requestByService(serviceId)(l, ctx)
				if err != nil {
					log.Fatalf("Could not retrieve configuration.")
				}
				serviceConfig = &c
			})
		}
	}
}
