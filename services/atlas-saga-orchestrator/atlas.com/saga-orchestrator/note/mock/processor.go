package mock

import (
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the note.Processor interface
type ProcessorMock struct {
	CreateNoteFunc func(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte, giftNote bool) error
}

// CreateNote is a mock implementation of the note.Processor.CreateNote method
func (m *ProcessorMock) CreateNote(transactionId uuid.UUID, receiverId uint32, senderId uint32, message string, flag byte, giftNote bool) error {
	if m.CreateNoteFunc != nil {
		return m.CreateNoteFunc(transactionId, receiverId, senderId, message, flag, giftNote)
	}
	return nil
}
