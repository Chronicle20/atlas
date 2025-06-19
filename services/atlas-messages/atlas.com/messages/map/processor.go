package _map

import (
	"atlas-messages/character"
	"atlas-messages/data/map"
	"atlas-messages/data/portal"
	character2 "atlas-messages/kafka/message/character"
	"atlas-messages/kafka/producer"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"strconv"
)

type Processor interface {
	Exists(mapId uint32) bool
	CharacterIdsInMapStringProvider(worldId byte, channelId byte, mapStr string) model.Provider[[]uint32]
	CharacterIdsInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32]
	WarpRandom(worldId byte) func(channelId byte) func(characterId uint32) func(mapId uint32) error
	WarpToPortal(worldId byte, channelId byte, characterId uint32, mapId uint32, pp model.Provider[uint32]) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	pp  portal.Processor
	dp  _map.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		pp:  portal.NewProcessor(l, ctx),
		dp:  _map.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) Exists(mapId uint32) bool {
	_, err := p.dp.GetById(mapId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find requested map [%d].", mapId)
		return false
	}
	return true
}

func (p *ProcessorImpl) CharacterIdsInMapStringProvider(worldId byte, channelId byte, mapStr string) model.Provider[[]uint32] {
	mapId, err := strconv.ParseUint(mapStr, 10, 32)
	if err != nil {
		return model.ErrorProvider[[]uint32](err)
	}
	return p.CharacterIdsInMapProvider(worldId, channelId, uint32(mapId))
}

func (p *ProcessorImpl) CharacterIdsInMapProvider(worldId byte, channelId byte, mapId uint32) model.Provider[[]uint32] {
	return requests.SliceProvider[RestModel, uint32](p.l, p.ctx)(requestCharactersInMap(worldId, channelId, mapId), Extract, model.Filters[uint32]())
}

func (p *ProcessorImpl) WarpRandom(worldId byte) func(channelId byte) func(characterId uint32) func(mapId uint32) error {
	return func(channelId byte) func(characterId uint32) func(mapId uint32) error {
		return func(characterId uint32) func(mapId uint32) error {
			return func(mapId uint32) error {
				return p.WarpToPortal(worldId, channelId, characterId, mapId, p.pp.RandomSpawnPointIdProvider(mapId))
			}
		}
	}
}

func (p *ProcessorImpl) WarpToPortal(worldId byte, channelId byte, characterId uint32, mapId uint32, pp model.Provider[uint32]) error {
	id, err := pp()
	if err != nil {
		return err
	}

	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(character.ChangeMapProvider(worldId, channelId, characterId, mapId, id))
}
