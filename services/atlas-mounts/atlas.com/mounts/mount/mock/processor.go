package mock

import (
	"atlas-mounts/kafka/message"
	"atlas-mounts/mount"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ProcessorMock struct {
	WithFunc             func(opts ...mount.ProcessorOption) mount.Processor
	GetByCharacterIdFunc func(characterId uint32) (mount.Model, error)
	ApplyTickFunc        func(mb *message.Buffer) func(worldId world.Id, characterId uint32) error
	ApplyFeedAndEmitFunc func(mb *message.Buffer) func(worldId world.Id, characterId uint32, healMax int) error
	EmitSetFunc          func(mb *message.Buffer) func(worldId world.Id, characterId uint32) error
}

var _ mount.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) With(opts ...mount.ProcessorOption) mount.Processor {
	if m.WithFunc != nil {
		return m.WithFunc(opts...)
	}
	return m
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) (mount.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return mount.Model{}, nil
}

func (m *ProcessorMock) ApplyTick(mb *message.Buffer) func(worldId world.Id, characterId uint32) error {
	if m.ApplyTickFunc != nil {
		return m.ApplyTickFunc(mb)
	}
	return func(worldId world.Id, characterId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) ApplyFeedAndEmit(mb *message.Buffer) func(worldId world.Id, characterId uint32, healMax int) error {
	if m.ApplyFeedAndEmitFunc != nil {
		return m.ApplyFeedAndEmitFunc(mb)
	}
	return func(worldId world.Id, characterId uint32, healMax int) error {
		return nil
	}
}

func (m *ProcessorMock) EmitSet(mb *message.Buffer) func(worldId world.Id, characterId uint32) error {
	if m.EmitSetFunc != nil {
		return m.EmitSetFunc(mb)
	}
	return func(worldId world.Id, characterId uint32) error {
		return nil
	}
}
