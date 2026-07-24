package _map

import (
	"atlas-channel/session"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	sp  session.Processor
}

type Processor interface {
	CharacterIdsInMapModelProvider(f field.Model) model.Provider[[]uint32]
	GetCharacterIdsInMap(f field.Model) ([]uint32, error)
	ForSessionsInSessionsMap(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model]
	ForSessionsInMap(f field.Model, o model.Operator[session.Model]) error
	CharacterIdsInMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]uint32]
	ForSessionsInMapAllInstances(worldId world.Id, channelId channel.Id, mapId _map.Id, o model.Operator[session.Model]) error
	OtherCharacterIdsInMapModelProvider(f field.Model, referenceCharacterId uint32) model.Provider[[]uint32]
	ForOtherSessionsInMap(f field.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		sp:  session.NewProcessor(l, ctx),
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) CharacterIdsInMapModelProvider(f field.Model) model.Provider[[]uint32] {
	return characterIds(p.sp.InFieldModelProvider(f))
}

func (p *ProcessorImpl) GetCharacterIdsInMap(f field.Model) ([]uint32, error) {
	return p.CharacterIdsInMapModelProvider(f)()
}

func (p *ProcessorImpl) ForSessionsInSessionsMap(f func(oid uint32) model.Operator[session.Model]) model.Operator[session.Model] {
	return func(s session.Model) error {
		return p.sp.ForEachByCharacterId(s.Field().Channel())(p.CharacterIdsInMapModelProvider(s.Field()), f(s.CharacterId()))
	}
}

func (p *ProcessorImpl) ForSessionsInMap(f field.Model, o model.Operator[session.Model]) error {
	return p.sp.ForEachByCharacterId(f.Channel())(p.CharacterIdsInMapModelProvider(f), o)
}

func (p *ProcessorImpl) CharacterIdsInMapAllInstancesModelProvider(worldId world.Id, channelId channel.Id, mapId _map.Id) model.Provider[[]uint32] {
	return characterIds(p.sp.InMapAllInstancesModelProvider(worldId, channelId, mapId))
}

// characterIds maps sessions to their character ids, deduplicated — the
// registry can transiently hold two sessions for one character (stale socket
// plus reconnect) and each character must be delivered to at most once.
func characterIds(sp model.Provider[[]session.Model]) model.Provider[[]uint32] {
	return func() ([]uint32, error) {
		ss, err := sp()
		if err != nil {
			return nil, err
		}
		seen := make(map[uint32]struct{}, len(ss))
		ids := make([]uint32, 0, len(ss))
		for _, s := range ss {
			id := s.CharacterId()
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		return ids, nil
	}
}

func (p *ProcessorImpl) ForSessionsInMapAllInstances(worldId world.Id, channelId channel.Id, mapId _map.Id, o model.Operator[session.Model]) error {
	return p.sp.ForEachByCharacterId(channel.NewModel(worldId, channelId))(p.CharacterIdsInMapAllInstancesModelProvider(worldId, channelId, mapId), o)
}

func NotCharacterIdFilter(referenceCharacterId uint32) func(characterId uint32) bool {
	return func(characterId uint32) bool {
		return referenceCharacterId != characterId
	}
}

func (p *ProcessorImpl) OtherCharacterIdsInMapModelProvider(f field.Model, referenceCharacterId uint32) model.Provider[[]uint32] {
	return model.FilteredProvider(p.CharacterIdsInMapModelProvider(f), model.Filters(NotCharacterIdFilter(referenceCharacterId)))
}

func (p *ProcessorImpl) ForOtherSessionsInMap(f field.Model, referenceCharacterId uint32, o model.Operator[session.Model]) error {
	mp := p.OtherCharacterIdsInMapModelProvider(f, referenceCharacterId)
	return p.sp.ForEachByCharacterId(f.Channel())(mp, o)
}
