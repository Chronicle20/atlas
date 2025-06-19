package skill

import (
	skill3 "atlas-channel/data/skill"
	skill2 "atlas-channel/kafka/message/skill"
	"atlas-channel/kafka/producer"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/skill"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
	}
	return p
}

func (p *Processor) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
}

func (p *Processor) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *Processor) ApplyCooldown(m _map.Model, skillId skill.Id, cooldown uint32) model.Operator[uint32] {
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
