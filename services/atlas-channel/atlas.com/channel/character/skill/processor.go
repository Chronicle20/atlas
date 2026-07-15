package skill

import (
	skill3 "atlas-channel/data/skill"
	skill2 "atlas-channel/kafka/message/skill"
	"context"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for skill processing
type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	ApplyCooldown(f field.Model, skillId skill.Id, cooldown uint32) model.Operator[uint32]
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
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(characterSkillsUrl(characterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) ApplyCooldown(_ field.Model, skillId skill.Id, cooldown uint32) model.Operator[uint32] {
	return func(characterId uint32) error {
		return producer.ProviderImpl(p.l)(p.ctx)(skill2.EnvCommandTopic)(skill3.SetCooldownCommandProvider(characterId, uint32(skillId), cooldown))
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
