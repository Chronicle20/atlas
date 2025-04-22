package _map

import (
	"atlas-channel/session"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	sp  *session.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		sp:  session.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) CharacterIdsInMapModelProvider(m _map.Model) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMap(m), Extract, model.Filters[uint32]())
}

func (p *Processor) GetCharacterIdsInMap(m _map.Model) ([]uint32, error) {
	return p.CharacterIdsInMapModelProvider(m)()
}

func (p *Processor) ForSessionsInSessionsMap(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model] {
	return func(s session.Model) error {
		return p.sp.ForEachByCharacterId(s.WorldId(), s.ChannelId())(p.CharacterIdsInMapModelProvider(s.Map()), f(s.CharacterId()))
	}
}

func (p *Processor) ForSessionsInMap(m _map.Model, o model.Operator[session.Model]) error {
	return p.sp.ForEachByCharacterId(m.WorldId(), m.ChannelId())(p.CharacterIdsInMapModelProvider(m), o)
}

func NotCharacterIdFilter(referenceCharacterId uint32) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		return referenceCharacterId != characterId
	}
}

func (p *Processor) OtherCharacterIdsInMapModelProvider(m _map.Model, referenceCharacterId uint32) model.Provider[[]uint32] {
	return model.FilteredProvider(p.CharacterIdsInMapModelProvider(m), model.Filters(NotCharacterIdFilter(referenceCharacterId)))
}

func (p *Processor) ForOtherSessionsInMap(m _map.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error {
	mp := p.OtherCharacterIdsInMapModelProvider(m, referenceCharacterId)
	return p.sp.ForEachByCharacterId(m.WorldId(), m.ChannelId())(mp, o)
}
