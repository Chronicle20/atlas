package quest

import (
	"atlas-channel/kafka/message/quest"
	"atlas-channel/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	StartQuestConversation(f field.Model, questId uint32, npcId uint32, characterId uint32) error
	StartQuest(f field.Model, characterId uint32, questId uint32, npcId uint32, force bool) error
	CompleteQuest(f field.Model, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) error
	ForfeitQuest(f field.Model, characterId uint32, questId uint32) error
	RestoreItem(f field.Model, characterId uint32, questId uint32, itemId uint32) error
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
}

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

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacterId(characterId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) StartQuestConversation(f field.Model, questId uint32, npcId uint32, characterId uint32) error {
	p.l.Debugf("Starting quest [%d] conversation for character [%d] with NPC [%d].", questId, characterId, npcId)
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvCommandTopic)(StartConversationCommandProvider(f, questId, npcId, characterId))
}

func (p *ProcessorImpl) StartQuest(f field.Model, characterId uint32, questId uint32, npcId uint32, force bool) error {
	p.l.Debugf("Sending start quest [%d] command for character [%d] with NPC [%d]. force [%t]", questId, characterId, npcId, force)
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvQuestCommandTopic)(StartQuestCommandProvider(f, characterId, questId, npcId, force))
}

func (p *ProcessorImpl) CompleteQuest(f field.Model, characterId uint32, questId uint32, npcId uint32, selection int32, force bool) error {
	p.l.Debugf("Sending complete quest [%d] command for character [%d] with NPC [%d]. selection [%d] force [%t]", questId, characterId, npcId, selection, force)
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvQuestCommandTopic)(CompleteQuestCommandProvider(f, characterId, questId, npcId, selection, force))
}

func (p *ProcessorImpl) ForfeitQuest(f field.Model, characterId uint32, questId uint32) error {
	p.l.Debugf("Sending forfeit quest [%d] command for character [%d].", questId, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvQuestCommandTopic)(ForfeitQuestCommandProvider(f, characterId, questId))
}

func (p *ProcessorImpl) RestoreItem(f field.Model, characterId uint32, questId uint32, itemId uint32) error {
	p.l.Debugf("Sending restore item [%d] for quest [%d] command for character [%d].", itemId, questId, characterId)
	return producer.ProviderImpl(p.l)(p.ctx)(quest.EnvQuestCommandTopic)(RestoreItemCommandProvider(f, characterId, questId, itemId))
}
