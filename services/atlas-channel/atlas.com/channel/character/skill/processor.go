package skill

import (
	skill3 "atlas-channel/data/skill"
	skill2 "atlas-channel/kafka/message/skill"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor interface defines the operations for skill processing
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	ApplyCooldown(f field.Model, skillId skill.Id, cooldown uint32) model.Operator[uint32]
	ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// ByCharacterIdProvider fetches every skill for a character. The upstream
// atlas-skills list is now paginated (task-117); callers here need the
// complete set (e.g. sending the full skill record on channel spawn), so
// this drains every page rather than fetching just the first.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	url, err := characterSkillsUrl(p.ctx, characterId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) ApplyCooldown(_ field.Model, skillId skill.Id, cooldown uint32) model.Operator[uint32] {
	return func(characterId uint32) error {
		return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(skill3.SetCooldownCommandProvider(characterId, uint32(skillId), cooldown))
	}
}

// ResetCooldowns emits a RESET_COOLDOWNS command for the operated-on
// character, clearing every cooldown except exceptSkillIds. Mirrors
// ApplyCooldown's operator shape so callers can fan out over recipients.
func (p *ProcessorImpl) ResetCooldowns(transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32) model.Operator[uint32] {
	return func(characterId uint32) error {
		return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(skill3.ResetCooldownsCommandProvider(transactionId, f.WorldId(), characterId, exceptSkillIds, sourceSkillId))
	}
}

func GetLevel(skills []Model, id skill.Id) byte {
	for _, s := range skills {
		if s.Id() == id {
			return s.Level()
		}
	}
	return 0
}

// GetLevelIdentity is the Identity form of GetLevel (task-187): it resolves
// each trained skill's wire id through set before comparing against
// identity, rather than comparing raw wire ids directly. Callers that
// already hold a version-blind Identity constant (rather than a raw wire
// id) use this instead of GetLevel.
func GetLevelIdentity(skills []Model, set skill.Set, identity skill.Identity) byte {
	for _, s := range skills {
		if resolved, ok := set.Resolve(s.Id()); ok && resolved == identity {
			return s.Level()
		}
	}
	return 0
}
