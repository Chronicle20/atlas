package data

import (
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
