package note

import (
	note2 "atlas-saga-orchestrator/kafka/message/note"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Processor is the interface for note operations
type Processor interface {
	CreateNote(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte, giftNote bool) error
}

// ProcessorImpl is the implementation of the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new note processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// CreateNote emits the note CREATE command for the create_note saga step.
// The step completes when atlas-notes' CREATED/CREATE_FAILED status event
// arrives (kafka/consumer/note/consumer.go).
func (p *ProcessorImpl) CreateNote(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte, giftNote bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(note2.EnvCommandTopic)(CreateNoteCommandProvider(transactionId, receiverId, senderId, message, flag, giftNote))
}
