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
	ErrorCodeNotInParty         = "PQ_NOT_IN_PARTY"
	ErrorCodeNotLeader          = "PQ_NOT_LEADER"
	ErrorCodePartySizeFailed    = "PQ_PARTY_SIZE"
	ErrorCodeLevelMinFailed     = "PQ_LEVEL_MIN"
	ErrorCodeLevelMaxFailed     = "PQ_LEVEL_MAX"
	ErrorCodeDefinitionNotFound = "PQ_DEFINITION_NOT_FOUND"
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
	GetPartyMembers(characterId uint32) ([]MemberRestModel, error)
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

	// Fetch the party quest definition to validate start requirements
	definition, err := p.getDefinition(questId)
	if err != nil {
		return PartyQuestError{
			Code:    ErrorCodeDefinitionNotFound,
			Message: fmt.Sprintf("failed to get definition for quest %s: %s", questId, err.Error()),
		}
	}

	// Validate start requirements if any are defined
	if len(definition.StartRequirements) > 0 {
		members, err := p.getPartyMembers(party.Id)
		if err != nil {
			return fmt.Errorf("failed to get party members for party %d: %w", party.Id, err)
		}

		if err := validateStartRequirements(definition.StartRequirements, members); err != nil {
			return err
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

// getDefinition fetches the party quest definition by questId from atlas-party-quests
func (p *ProcessorImpl) getDefinition(questId string) (DefinitionRestModel, error) {
	return requests.Provider[DefinitionRestModel, DefinitionRestModel](p.l, p.ctx)(requestDefinitionByQuestId(questId), ExtractDefinition)()
}

// getPartyMembers fetches all members of a party from atlas-parties
func (p *ProcessorImpl) getPartyMembers(partyId uint32) ([]MemberRestModel, error) {
	return requests.SliceProvider[MemberRestModel, MemberRestModel](p.l, p.ctx)(requestPartyMembers(partyId), ExtractMember, model.Filters[MemberRestModel]())()
}

// GetPartyMembers resolves the party for a character and returns all party members.
func (p *ProcessorImpl) GetPartyMembers(characterId uint32) ([]MemberRestModel, error) {
	party, err := p.getParty(characterId)
	if err != nil {
		return nil, PartyQuestError{
			Code:    ErrorCodeNotInParty,
			Message: fmt.Sprintf("failed to get party for character %d: %s", characterId, err.Error()),
		}
	}

	if party.Id == 0 {
		return nil, PartyQuestError{
			Code:    ErrorCodeNotInParty,
			Message: fmt.Sprintf("character %d is not in a party", characterId),
		}
	}

	members, err := p.getPartyMembers(party.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get party members for party %d: %w", party.Id, err)
	}

	return members, nil
}

// validateStartRequirements checks the party quest definition's startRequirements
// against the actual party members. Only party_size, level_min, and level_max
// are validated. Returns nil if all requirements are met or if requirements is empty.
func validateStartRequirements(requirements []ConditionRestModel, members []MemberRestModel) error {
	for _, req := range requirements {
		switch req.Type {
		case "party_size":
			if !compareUint32(uint32(len(members)), req.Operator, req.Value) {
				return PartyQuestError{
					Code:    ErrorCodePartySizeFailed,
					Message: fmt.Sprintf("party size %d does not satisfy %s %d", len(members), req.Operator, req.Value),
				}
			}
		case "level_min":
			for _, m := range members {
				if !compareUint32(uint32(m.Level), req.Operator, req.Value) {
					return PartyQuestError{
						Code:    ErrorCodeLevelMinFailed,
						Message: fmt.Sprintf("member %d level %d does not satisfy level_min %s %d", m.Id, m.Level, req.Operator, req.Value),
					}
				}
			}
		case "level_max":
			for _, m := range members {
				if !compareUint32(uint32(m.Level), req.Operator, req.Value) {
					return PartyQuestError{
						Code:    ErrorCodeLevelMaxFailed,
						Message: fmt.Sprintf("member %d level %d does not satisfy level_max %s %d", m.Id, m.Level, req.Operator, req.Value),
					}
				}
			}
		default:
			continue
		}
	}
	return nil
}

// compareUint32 evaluates: actual <operator> expected
func compareUint32(actual uint32, operator string, expected uint32) bool {
	switch operator {
	case "eq":
		return actual == expected
	case "gte":
		return actual >= expected
	case "lte":
		return actual <= expected
	case "gt":
		return actual > expected
	case "lt":
		return actual < expected
	default:
		return false
	}
}
