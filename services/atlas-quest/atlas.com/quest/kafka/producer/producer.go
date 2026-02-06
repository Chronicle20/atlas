package producer

import (
	"github.com/Chronicle20/atlas-kafka/producer"
)

type Provider func(token string) producer.MessageProducer
