package thread

import (
	thread2 "atlas-channel/kafka/message/guild/thread"
	"atlas-channel/kafka/producer"
	"context"

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

func (p *Processor) GetById(guildId uint32, threadId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(guildId, threadId), Extract)()
}

func (p *Processor) GetAll(guildId uint32) ([]Model, error) {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestAll(guildId), Extract, model.Filters[Model]())()
}

func (p *Processor) ModifyThread(guildId uint32, characterId uint32, threadId uint32, notice bool, title string, message string, emoticonId uint32) error {
	p.l.Debugf("Character [%d] is attempting to modify a guild thread [%d] to have, notice [%t], title [%s], message [%s], emoticonId [%d].", characterId, threadId, notice, title, message, emoticonId)
	return producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvCommandTopic)(UpdateCommandProvider(guildId, characterId, threadId, notice, title, message, emoticonId))
}

func (p *Processor) CreateThread(guildId uint32, characterId uint32, notice bool, title string, message string, emoticonId uint32) error {
	p.l.Debugf("Character [%d] is attempting to create a guild thread to have, notice [%t], title [%s], message [%s], emoticonId [%d].", characterId, notice, title, message, emoticonId)
	return producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvCommandTopic)(CreateCommandProvider(guildId, characterId, notice, title, message, emoticonId))
}

func (p *Processor) DeleteThread(guildId uint32, characterId uint32, threadId uint32) error {
	p.l.Debugf("Character [%d] attempting to delete guild thread [%d].", characterId, threadId)
	return producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvCommandTopic)(DeleteCommandProvider(guildId, characterId, threadId))
}

func (p *Processor) ListThreads(guildId uint32, characterId uint32, startIndex uint32) error {
	p.l.Debugf("Character [%d] attempting list threads starting at [%d].", characterId, startIndex)
	return nil
}

func (p *Processor) ReplyToThread(guildId uint32, characterId uint32, threadId uint32, message string) error {
	p.l.Debugf("Character [%d] attempting to reply to guild thread [%d] with message [%s].", characterId, threadId, message)
	return producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvCommandTopic)(AddReplyCommandProvider(guildId, characterId, threadId, message))
}

func (p *Processor) DeleteReply(guildId uint32, characterId uint32, threadId uint32, replyId uint32) error {
	p.l.Debugf("Character [%d] attempting to delete reply [%d] in guild thread [%d].", characterId, replyId, threadId)
	return producer.ProviderImpl(p.l)(p.ctx)(thread2.EnvCommandTopic)(DeleteReplyCommandProvider(guildId, characterId, threadId, replyId))
}
