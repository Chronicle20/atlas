package system_message

import (
	"atlas-saga-orchestrator/kafka/message/system_message"
	"atlas-saga-orchestrator/kafka/producer"
	"context"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for system message operations
type Processor interface {
	// SendMessage sends a system message to a character
	SendMessage(transactionId uuid.UUID, ch channel.Model, characterId uint32, messageType string, message string) error
	// PlayPortalSound sends a command to play the portal sound effect for a character
	PlayPortalSound(transactionId uuid.UUID, ch channel.Model, characterId uint32) error
	// ShowInfo sends a command to show an info/tutorial effect for a character
	ShowInfo(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error
	// ShowInfoText sends a command to show a text message for a character
	ShowInfoText(transactionId uuid.UUID, ch channel.Model, characterId uint32, text string) error
	// UpdateAreaInfo sends a command to update area info (quest record ex) for a character
	UpdateAreaInfo(transactionId uuid.UUID, ch channel.Model, characterId uint32, area uint16, info string) error
	// ShowHint sends a command to show a hint box for a character
	ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error
	// ShowGuideHint sends a command to show a pre-defined guide hint by ID for a character
	ShowGuideHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hintId uint32, duration uint32) error
	// ShowIntro sends a command to show an intro/direction effect for a character
	ShowIntro(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new system message processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// SendMessage sends a Kafka command to atlas-channel to display a system message
func (p *ProcessorImpl) SendMessage(transactionId uuid.UUID, ch channel.Model, characterId uint32, messageType string, message string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(SendMessageCommandProvider(transactionId, ch, characterId, messageType, message))
}

// PlayPortalSound sends a Kafka command to atlas-channel to play the portal sound effect
func (p *ProcessorImpl) PlayPortalSound(transactionId uuid.UUID, ch channel.Model, characterId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(PlayPortalSoundCommandProvider(transactionId, ch, characterId))
}

// ShowInfo sends a Kafka command to atlas-channel to show an info/tutorial effect
func (p *ProcessorImpl) ShowInfo(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(ShowInfoCommandProvider(transactionId, ch, characterId, path))
}

// ShowInfoText sends a Kafka command to atlas-channel to show a text message
func (p *ProcessorImpl) ShowInfoText(transactionId uuid.UUID, ch channel.Model, characterId uint32, text string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(ShowInfoTextCommandProvider(transactionId, ch, characterId, text))
}

// UpdateAreaInfo sends a Kafka command to atlas-channel to update area info
func (p *ProcessorImpl) UpdateAreaInfo(transactionId uuid.UUID, ch channel.Model, characterId uint32, area uint16, info string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(UpdateAreaInfoCommandProvider(transactionId, ch, characterId, area, info))
}

// ShowHint sends a Kafka command to atlas-channel to show a hint box
func (p *ProcessorImpl) ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(ShowHintCommandProvider(transactionId, ch, characterId, hint, width, height))
}

// ShowGuideHint sends a Kafka command to atlas-channel to show a pre-defined guide hint by ID
func (p *ProcessorImpl) ShowGuideHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hintId uint32, duration uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(ShowGuideHintCommandProvider(transactionId, ch, characterId, hintId, duration))
}

// ShowIntro sends a Kafka command to atlas-channel to show an intro/direction effect
func (p *ProcessorImpl) ShowIntro(transactionId uuid.UUID, ch channel.Model, characterId uint32, path string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(system_message.EnvCommandTopic)(ShowIntroCommandProvider(transactionId, ch, characterId, path))
}
