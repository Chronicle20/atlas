package mock

import (
	"atlas-notes/kafka/message"
	"atlas-notes/note"
	"github.com/Chronicle20/atlas-model/model"
)

type ProcessorMock struct {
	CreateFunc              func(mb *message.Buffer) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (note.Model, error)
	CreateAndEmitFunc       func(characterId uint32, senderId uint32, msg string, flag byte) (note.Model, error)
	UpdateFunc              func(mb *message.Buffer) func(id uint32) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (note.Model, error)
	UpdateAndEmitFunc       func(id uint32, characterId uint32, senderId uint32, msg string, flag byte) (note.Model, error)
	DeleteFunc              func(mb *message.Buffer) func(id uint32) error
	DeleteAndEmitFunc       func(id uint32) error
	DeleteAllFunc           func(mb *message.Buffer) func(characterId uint32) error
	DeleteAllAndEmitFunc    func(characterId uint32) error
	DiscardFunc             func(mb *message.Buffer) func(characterId uint32) func(noteIds []uint32) error
	DiscardAndEmitFunc      func(characterId uint32, noteIds []uint32) error
	ByIdProviderFunc        func(id uint32) model.Provider[note.Model]
	ByCharacterProviderFunc func(characterId uint32) model.Provider[[]note.Model]
	InTenantProviderFunc    func() model.Provider[[]note.Model]
}

func (m *ProcessorMock) Create(mb *message.Buffer) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (note.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(mb)
	}
	return func(uint32) func(uint32) func(string) func(byte) (note.Model, error) {
		return func(uint32) func(string) func(byte) (note.Model, error) {
			return func(string) func(byte) (note.Model, error) {
				return func(byte) (note.Model, error) {
					return note.Model{}, nil
				}
			}
		}
	}
}

func (m *ProcessorMock) CreateAndEmit(characterId uint32, senderId uint32, msg string, flag byte) (note.Model, error) {
	if m.CreateAndEmitFunc != nil {
		return m.CreateAndEmitFunc(characterId, senderId, msg, flag)
	}
	return note.Model{}, nil
}

func (m *ProcessorMock) Update(mb *message.Buffer) func(id uint32) func(characterId uint32) func(senderId uint32) func(msg string) func(flag byte) (note.Model, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(mb)
	}
	return func(uint32) func(uint32) func(uint32) func(string) func(byte) (note.Model, error) {
		return func(uint32) func(uint32) func(string) func(byte) (note.Model, error) {
			return func(uint32) func(string) func(byte) (note.Model, error) {
				return func(string) func(byte) (note.Model, error) {
					return func(byte) (note.Model, error) {
						return note.Model{}, nil
					}
				}
			}
		}
	}
}

func (m *ProcessorMock) UpdateAndEmit(id uint32, characterId uint32, senderId uint32, msg string, flag byte) (note.Model, error) {
	if m.UpdateAndEmitFunc != nil {
		return m.UpdateAndEmitFunc(id, characterId, senderId, msg, flag)
	}
	return note.Model{}, nil
}

func (m *ProcessorMock) Delete(mb *message.Buffer) func(id uint32) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(mb)
	}
	return func(id uint32) error {
		return nil
	}
}

func (m *ProcessorMock) DeleteAndEmit(id uint32) error {
	if m.DeleteAndEmitFunc != nil {
		return m.DeleteAndEmitFunc(id)
	}
	return nil
}

func (m *ProcessorMock) DeleteAll(mb *message.Buffer) func(characterId uint32) error {
	if m.DeleteAllFunc != nil {
		return m.DeleteAllFunc(mb)
	}
	return func(characterId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) DeleteAllAndEmit(characterId uint32) error {
	if m.DeleteAllAndEmitFunc != nil {
		return m.DeleteAllAndEmitFunc(characterId)
	}
	return nil
}

func (m *ProcessorMock) Discard(mb *message.Buffer) func(characterId uint32) func(noteIds []uint32) error {
	if m.DiscardFunc != nil {
		return m.DiscardFunc(mb)
	}
	return func(characterId uint32) func(noteIds []uint32) error {
		return func(noteIds []uint32) error {
			return nil
		}
	}
}

func (m *ProcessorMock) DiscardAndEmit(characterId uint32, noteIds []uint32) error {
	if m.DiscardAndEmitFunc != nil {
		return m.DiscardAndEmitFunc(characterId, noteIds)
	}
	return nil
}

func (m *ProcessorMock) ByIdProvider(id uint32) model.Provider[note.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return model.FixedProvider(note.Model{})
}

func (m *ProcessorMock) ByCharacterProvider(characterId uint32) model.Provider[[]note.Model] {
	if m.ByCharacterProviderFunc != nil {
		return m.ByCharacterProviderFunc(characterId)
	}
	return model.FixedProvider([]note.Model{})
}

func (m *ProcessorMock) InTenantProvider() model.Provider[[]note.Model] {
	if m.InTenantProviderFunc != nil {
		return m.InTenantProviderFunc()
	}
	return model.FixedProvider([]note.Model{})
}
