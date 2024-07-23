package producer

import (
	"encoding/json"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func emptyHeaders() ([]kafka.Header, error) {
	return make([]kafka.Header, 0), nil
}

func headerFolder(headers []kafka.Header, decorator HeaderDecorator) ([]kafka.Header, error) {
	hm, err := decorator()
	if err != nil {
		return nil, err
	}
	for k, v := range hm {
		headers = append(headers, kafka.Header{Key: k, Value: []byte(v)})
	}
	return headers, nil
}

func produceHeaders(decorators ...HeaderDecorator) ([]kafka.Header, error) {
	return model.Fold(model.FixedSliceProvider(decorators), emptyHeaders, headerFolder)()
}

type RawMessage struct {
	Key   []byte
	Value interface{}
}

func MessageProvider(mp model.SliceProvider[RawMessage]) model.SliceProvider[kafka.Message] {
	return model.SliceMap(mp, transformer, model.ParallelMap())
}

func SingleMessageProvider(key []byte, value interface{}) model.SliceProvider[kafka.Message] {
	return MessageProvider(model.FixedSingleSliceProvider(RawMessage{Key: key, Value: value}))
}

func transformer(rm RawMessage) (kafka.Message, error) {
	var value []byte
	value, err := json.Marshal(rm.Value)
	if err != nil {
		return kafka.Message{}, err
	}

	m := kafka.Message{Key: rm.Key, Value: value}
	//m.Headers, err = produceHeaders(decorators...)
	return m, nil
}

type MessageProducer func(provider model.SliceProvider[kafka.Message]) error

func ErrMessageProducer(err error) MessageProducer {
	return func(provider model.SliceProvider[kafka.Message]) error {
		return err
	}
}
