package family

import (
	"encoding/json"
	"time"

	"atlas-family/kafka/message/family"
	"atlas-family/kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// EventProducerImpl implements the EventProducer interface
type EventProducerImpl struct {
	log      logrus.FieldLogger
	producer producer.Provider
}

// NewEventProducer creates a new event producer instance
func NewEventProducer(log logrus.FieldLogger, producer producer.Provider) EventProducer {
	return &EventProducerImpl{
		log:      log,
		producer: producer,
	}
}

// LinkCreatedEventProvider creates a Kafka message provider for link created events
func (p *EventProducerImpl) LinkCreatedEventProvider(transactionId string, characterId uint32, seniorId uint32, juniorId uint32) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.NewLinkCreatedEvent(transactionId, characterId, seniorId, juniorId)
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal link created event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeLinkCreated)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// LinkBrokenEventProvider creates a Kafka message provider for link broken events
func (p *EventProducerImpl) LinkBrokenEventProvider(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, reason string) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.NewLinkBrokenEvent(transactionId, characterId, seniorId, juniorId, reason)
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal link broken event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeLinkBroken)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// RepGainedEventProvider creates a Kafka message provider for reputation gained events
func (p *EventProducerImpl) RepGainedEventProvider(transactionId string, characterId uint32, repGained uint32, dailyRep uint32, source string) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.NewRepGainedEvent(transactionId, characterId, repGained, dailyRep, source)
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal rep gained event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeRepGained)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// RepRedeemedEventProvider creates a Kafka message provider for reputation redeemed events
func (p *EventProducerImpl) RepRedeemedEventProvider(transactionId string, characterId uint32, repRedeemed uint32, reason string) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.NewRepRedeemedEvent(transactionId, characterId, repRedeemed, reason)
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal rep redeemed event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeRepRedeemed)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// RepErrorEventProvider creates a Kafka message provider for reputation error events
func (p *EventProducerImpl) RepErrorEventProvider(transactionId string, characterId uint32, errorCode string, errorMessage string, amount uint32) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.NewRepErrorEvent(transactionId, characterId, errorCode, errorMessage, amount)
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal rep error event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeRepError)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// LinkErrorEventProvider creates a Kafka message provider for link error events
func (p *EventProducerImpl) LinkErrorEventProvider(transactionId string, characterId uint32, seniorId uint32, juniorId uint32, errorCode string, errorMessage string) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.NewLinkErrorEvent(transactionId, characterId, seniorId, juniorId, errorCode, errorMessage)
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal link error event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeLinkError)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// Additional event providers for comprehensive family system support

// TreeDissolvedEventProvider creates a Kafka message provider for tree dissolved events
func (p *EventProducerImpl) TreeDissolvedEventProvider(transactionId string, characterId uint32, seniorId uint32, affectedIds []uint32, reason string) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.Event[family.TreeDissolvedEventBody]{
			TransactionId: transactionId,
			CharacterId:   characterId,
			Type:          family.EventTypeTreeDissolved,
			Body: family.TreeDissolvedEventBody{
				SeniorId:    seniorId,
				AffectedIds: affectedIds,
				Reason:      reason,
				Timestamp:   time.Now(),
			},
		}
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal tree dissolved event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeTreeDissolved)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// RepResetEventProvider creates a Kafka message provider for reputation reset events
func (p *EventProducerImpl) RepResetEventProvider(transactionId string, characterId uint32, previousDailyRep uint32) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.Event[family.RepResetEventBody]{
			TransactionId: transactionId,
			CharacterId:   characterId,
			Type:          family.EventTypeRepReset,
			Body: family.RepResetEventBody{
				PreviousDailyRep: previousDailyRep,
				Timestamp:        time.Now(),
			},
		}
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal rep reset event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeRepReset)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}

// RepCappedEventProvider creates a Kafka message provider for reputation capped events
func (p *EventProducerImpl) RepCappedEventProvider(transactionId string, characterId uint32, attemptedAmount uint32, dailyRep uint32, source string) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		event := family.Event[family.RepCappedEventBody]{
			TransactionId: transactionId,
			CharacterId:   characterId,
			Type:          family.EventTypeRepCapped,
			Body: family.RepCappedEventBody{
				AttemptedAmount: attemptedAmount,
				DailyRep:        dailyRep,
				Source:          source,
				Timestamp:       time.Now(),
			},
		}
		
		messageBody, err := json.Marshal(event)
		if err != nil {
			p.log.WithError(err).Error("Failed to marshal rep capped event")
			return []kafka.Message{}, err
		}

		message := kafka.Message{
			Key:   []byte(transactionId),
			Value: messageBody,
			Headers: []kafka.Header{
				{Key: "event_type", Value: []byte(family.EventTypeRepCapped)},
				{Key: "character_id", Value: []byte(string(rune(characterId)))},
				{Key: "transaction_id", Value: []byte(transactionId)},
			},
		}

		return []kafka.Message{message}, nil
	}
}