package npc

import (
	"atlas-channel/kafka/message/npc"
	"atlas-channel/kafka/producer"
	"atlas-channel/npc/shops"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	sp  shops.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		sp:  shops.NewProcessor(l, ctx),
	}
	return p
}

func (p *Processor) ForEachInMap(mapId _map.Id, f model.Operator[Model]) error {
	return model.ForEachSlice(p.InMapModelProvider(mapId), f, model.ParallelExecute())
}

func (p *Processor) InMapModelProvider(mapId _map.Id) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestNPCsInMap(mapId), Extract, model.Filters[Model]())
}

func (p *Processor) InMapByObjectIdModelProvider(mapId _map.Id, objectId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestNPCsInMapByObjectId(mapId, objectId), Extract, model.Filters[Model]())
}

func (p *Processor) GetInMapByObjectId(mapId _map.Id, objectId uint32) (Model, error) {
	mp := p.InMapByObjectIdModelProvider(mapId, objectId)
	return model.First[Model](mp, model.Filters[Model]())
}

func (p *Processor) StartConversation(m _map.Model, npcId uint32, characterId uint32) error {
	p.l.Debugf("Starting NPC [%d] conversation for character [%d].", npcId, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(npc.EnvCommandTopic)(StartConversationCommandProvider(m, npcId, characterId))
}

func (p *Processor) ContinueConversation(characterId uint32, action byte, lastMessageType byte, selection int32) error {
	p.l.Debugf("Continuing NPC conversation for character [%d]. action [%d], lastMessageType [%d], selection [%d].", characterId, action, lastMessageType, selection)
	return producer.ProviderImpl(p.l)(p.ctx)(npc.EnvCommandTopic)(ContinueConversationCommandProvider(characterId, action, lastMessageType, selection))
}

func (p *Processor) DisposeConversation(characterId uint32) error {
	p.l.Debugf("Ending NPC conversation for character [%d].", characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(npc.EnvCommandTopic)(DisposeConversationCommandProvider(characterId))
}

func (p *Processor) GetShop(template uint32) (shops.Model, error) {
	return p.sp.GetShop(template)
}
