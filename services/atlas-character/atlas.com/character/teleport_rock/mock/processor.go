package mock

import (
	"atlas-character/kafka/message"
	"atlas-character/teleport_rock"

	"github.com/google/uuid"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ProcessorMock is the func-field mock for teleport_rock.Processor (fame/notes
// convention — atlas-character's first standard mock).
type ProcessorMock struct {
	GetByCharacterIdFunc func(characterId uint32) (teleport_rock.Model, error)
	AddMapFunc           func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	AddMapAndEmitFunc    func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	AddFunc              func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (teleport_rock.Model, error)
	RemoveMapFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveMapAndEmitFunc func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error
	RemoveFunc           func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (teleport_rock.Model, error)
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) (teleport_rock.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return teleport_rock.Model{}, nil
}

func (m *ProcessorMock) AddMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.AddMapFunc != nil {
		return m.AddMapFunc(mb)
	}
	return func(uuid.UUID, world.Id, uint32, _map.Id, bool) error { return nil }
}

func (m *ProcessorMock) AddMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.AddMapAndEmitFunc != nil {
		return m.AddMapAndEmitFunc(transactionId, worldId, characterId, mapId, vip)
	}
	return nil
}

func (m *ProcessorMock) Add(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (teleport_rock.Model, error) {
	if m.AddFunc != nil {
		return m.AddFunc(transactionId, worldId, characterId, mapId, vip)
	}
	return teleport_rock.Model{}, nil
}

func (m *ProcessorMock) RemoveMap(mb *message.Buffer) func(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.RemoveMapFunc != nil {
		return m.RemoveMapFunc(mb)
	}
	return func(uuid.UUID, world.Id, uint32, _map.Id, bool) error { return nil }
}

func (m *ProcessorMock) RemoveMapAndEmit(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) error {
	if m.RemoveMapAndEmitFunc != nil {
		return m.RemoveMapAndEmitFunc(transactionId, worldId, characterId, mapId, vip)
	}
	return nil
}

func (m *ProcessorMock) Remove(transactionId uuid.UUID, worldId world.Id, characterId uint32, mapId _map.Id, vip bool) (teleport_rock.Model, error) {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(transactionId, worldId, characterId, mapId, vip)
	}
	return teleport_rock.Model{}, nil
}

var _ teleport_rock.Processor = (*ProcessorMock)(nil)
