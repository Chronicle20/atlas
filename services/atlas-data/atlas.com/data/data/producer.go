package data

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func startWorkerCommandProvider(name string, path string) model.Provider[[]kafka.Message] {
	value := &command[startWorkerCommandBody]{
		Type: CommandStartWorker,
		Body: startWorkerCommandBody{
			Name: name,
			Path: path,
		},
	}
	return producer.SingleMessageProvider(nil, value)
}

func dataUpdatedEventProvider(tenantId string, worker string, completedAt time.Time) model.Provider[[]kafka.Message] {
	key := []byte(tenantId)
	value := &event[dataUpdatedEventBody]{
		Type: EventTypeDataUpdated,
		Body: dataUpdatedEventBody{
			TenantId:    tenantId,
			Worker:      worker,
			CompletedAt: completedAt.UTC().Format(time.RFC3339),
		},
	}
	return producer.SingleMessageProvider(key, value)
}
