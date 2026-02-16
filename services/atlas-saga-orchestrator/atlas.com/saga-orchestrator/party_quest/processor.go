package party_quest

import (
	"context"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// Error codes for party quest registration failures
const (
	ErrorCodeNotInParty = "PQ_NOT_IN_PARTY"
	ErrorCodeNotLeader  = "PQ_NOT_LEADER"
)

// PartyQuestError represents an error from the party quest registration with an error code
type PartyQuestError struct {
	Code    string
	Message string
}

func (e PartyQuestError) Error() string {
	return e.Message
}

// GetErrorCode extracts the error code from a PartyQuestError, or returns a default for other errors
func GetErrorCode(err error) string {
	var pqErr PartyQuestError
	if errors.As(err, &pqErr) {
		return pqErr.Code
	}
	return "PQ_UNKNOWN"
}

// Processor is the interface for party quest operations
type Processor interface {
	RegisterPartyQuest(characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, questId string) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new party quest processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// RegisterPartyQuest validates party state and produces a REGISTER command to atlas-party-quests
func (p *ProcessorImpl) RegisterPartyQuest(characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, questId string) error {
	// Get the character's party
	party, err := p.getParty(characterId)
	if err != nil {
		return PartyQuestError{
			Code:    ErrorCodeNotInParty,
			Message: fmt.Sprintf("failed to get party for character %d: %s", characterId, err.Error()),
		}
	}

	// Check if character is in a party
	if party.Id == 0 {
		return PartyQuestError{
			Code:    ErrorCodeNotInParty,
			Message: fmt.Sprintf("character %d is not in a party", characterId),
		}
	}

	// Check if character is the party leader
	if party.LeaderId != characterId {
		return PartyQuestError{
			Code:    ErrorCodeNotLeader,
			Message: fmt.Sprintf("character %d is not the party leader", characterId),
		}
	}

	p.l.WithFields(logrus.Fields{
		"character_id": characterId,
		"party_id":     party.Id,
		"quest_id":     questId,
	}).Debug("Producing REGISTER command for party quest")

	// Produce REGISTER command to party-quests
	err = p.produceRegisterCommand(characterId, worldId, channelId, mapId, questId, party.Id)
	if err != nil {
		return fmt.Errorf("failed to produce REGISTER command: %w", err)
	}

	return nil
}

// getParty fetches the party for a character from atlas-parties
func (p *ProcessorImpl) getParty(characterId uint32) (PartyRestModel, error) {
	models, err := requests.SliceProvider[PartyRestModel, PartyRestModel](p.l, p.ctx)(requestPartyByMemberId(characterId), ExtractParty, model.Filters[PartyRestModel]())()
	if err != nil {
		return PartyRestModel{}, err
	}
	if len(models) == 0 {
		return PartyRestModel{}, nil
	}
	return models[0], nil
}

// produceRegisterCommand produces a REGISTER command to the party-quests command topic
func (p *ProcessorImpl) produceRegisterCommand(characterId uint32, worldId world.Id, channelId channel.Id, mapId _map.Id, questId string, partyId uint32) error {
	key := producer.CreateKey(int(characterId))
	value := &Command[RegisterCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        CommandTypeRegister,
		Body: RegisterCommandBody{
			QuestId:   questId,
			PartyId:   partyId,
			ChannelId: channelId,
			MapId:     uint32(mapId),
		},
	}
	mp := producer.SingleMessageProvider(key, value)
	return produceToCommandTopic(p.l, p.ctx)(mp)
}

// produceToCommandTopic produces messages to the party quest command topic
func produceToCommandTopic(l logrus.FieldLogger, ctx context.Context) func(provider model.Provider[[]kafka.Message]) error {
	sd := producer.SpanHeaderDecorator(ctx)
	td := producer.TenantHeaderDecorator(ctx)
	return producer.Produce(l)(producer.WriterProvider(topic.EnvProvider(l)(EnvCommandTopic)))(sd, td)
}
