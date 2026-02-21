package _map

import (
	"atlas-channel/session"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
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

func (p *Processor) CharacterIdsInMapModelProvider(f field.Model) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMap(f), Extract, model.Filters[uint32]())
}

func (p *Processor) GetCharacterIdsInMap(f field.Model) ([]uint32, error) {
	return p.CharacterIdsInMapModelProvider(f)()
}

func (p *Processor) ForSessionsInSessionsMap(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model] {
	return func(s session.Model) error {
		return p.sp.ForEachByCharacterId(s.Field().Channel())(p.CharacterIdsInMapModelProvider(s.Field()), f(s.CharacterId()))
	}
}

func (p *Processor) ForSessionsInMap(f field.Model, o model.Operator[session.Model]) error {
	return p.sp.ForEachByCharacterId(f.Channel())(p.CharacterIdsInMapModelProvider(f), o)
}

func (p *Processor) CharacterIdsInMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMapAllInstances(worldId, channelId, mapId), Extract, model.Filters[uint32]())
}

func (p *Processor) ForSessionsInMapAllInstances(worldId world.Id, channelId channel.Id, mapId _map.Id, o model.Operator[session.Model]) error {
	return p.sp.ForEachByCharacterId(channel.NewModel(worldId, channelId))(p.CharacterIdsInMapAllInstancesModelProvider(worldId, channelId, mapId), o)
}

func NotCharacterIdFilter(referenceCharacterId uint32) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		return referenceCharacterId != characterId
	}
}

func (p *Processor) OtherCharacterIdsInMapModelProvider(f field.Model, referenceCharacterId uint32) model.Provider[[]uint32] {
	return model.FilteredProvider(p.CharacterIdsInMapModelProvider(f), model.Filters(NotCharacterIdFilter(referenceCharacterId)))
}

func (p *Processor) ForOtherSessionsInMap(f field.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error {
	mp := p.OtherCharacterIdsInMapModelProvider(f, referenceCharacterId)
	return p.sp.ForEachByCharacterId(f.Channel())(mp, o)
}
