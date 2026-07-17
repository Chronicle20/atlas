package teleportrock

import (
	teleportrock2 "atlas-channel/kafka/message/teleportrock"
	"context"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	RequestAddMap(f field.Model, characterId uint32, vip bool) error
	RequestRemoveMap(worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract)()
}

// RequestAddMap registers the character's CURRENT map (server-derived from
// session state — the client sends no map id on register, design §1 Q1).
func (p *ProcessorImpl) RequestAddMap(f field.Model, characterId uint32, vip bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(teleportrock2.EnvCommandTopic)(addMapCommandProvider(uuid.New(), f.WorldId(), characterId, f.MapId(), vip))
}

func (p *ProcessorImpl) RequestRemoveMap(worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(teleportrock2.EnvCommandTopic)(removeMapCommandProvider(uuid.New(), worldId, characterId, mapId, vip))
}
