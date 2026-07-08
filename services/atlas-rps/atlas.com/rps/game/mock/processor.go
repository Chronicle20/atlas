package mock

import (
	"atlas-rps/game"
	"atlas-rps/kafka/message"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ProcessorMock is a test double for game.Processor. Each exported Func field
// is used when set; otherwise the method returns a zero Model and a nil
// error.
type ProcessorMock struct {
	GetFunc func(characterId uint32) (game.Model, game.Rung, bool, error)

	StartFunc        func(mb *message.Buffer, characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (game.Model, error)
	StartAndEmitFunc func(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (game.Model, error)

	SelectFunc        func(mb *message.Buffer, characterId uint32, throw game.Throw) (game.Model, error)
	SelectAndEmitFunc func(characterId uint32, throw game.Throw) (game.Model, error)

	ContinueFunc        func(mb *message.Buffer, characterId uint32) (game.Model, error)
	ContinueAndEmitFunc func(characterId uint32) (game.Model, error)

	CollectFunc        func(mb *message.Buffer, characterId uint32) (game.Model, error)
	CollectAndEmitFunc func(characterId uint32) (game.Model, error)

	QuitFunc        func(mb *message.Buffer, characterId uint32) (game.Model, error)
	QuitAndEmitFunc func(characterId uint32) (game.Model, error)

	DisposeFunc        func(mb *message.Buffer, characterId uint32) (game.Model, error)
	DisposeAndEmitFunc func(characterId uint32) (game.Model, error)
}

var _ game.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Get(characterId uint32) (game.Model, game.Rung, bool, error) {
	if m.GetFunc != nil {
		return m.GetFunc(characterId)
	}
	return game.Model{}, game.Rung{}, false, nil
}

func (m *ProcessorMock) Start(mb *message.Buffer, characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (game.Model, error) {
	if m.StartFunc != nil {
		return m.StartFunc(mb, characterId, worldId, channelId, npcId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) StartAndEmit(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (game.Model, error) {
	if m.StartAndEmitFunc != nil {
		return m.StartAndEmitFunc(characterId, worldId, channelId, npcId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) Select(mb *message.Buffer, characterId uint32, throw game.Throw) (game.Model, error) {
	if m.SelectFunc != nil {
		return m.SelectFunc(mb, characterId, throw)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) SelectAndEmit(characterId uint32, throw game.Throw) (game.Model, error) {
	if m.SelectAndEmitFunc != nil {
		return m.SelectAndEmitFunc(characterId, throw)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) Continue(mb *message.Buffer, characterId uint32) (game.Model, error) {
	if m.ContinueFunc != nil {
		return m.ContinueFunc(mb, characterId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) ContinueAndEmit(characterId uint32) (game.Model, error) {
	if m.ContinueAndEmitFunc != nil {
		return m.ContinueAndEmitFunc(characterId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) Collect(mb *message.Buffer, characterId uint32) (game.Model, error) {
	if m.CollectFunc != nil {
		return m.CollectFunc(mb, characterId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) CollectAndEmit(characterId uint32) (game.Model, error) {
	if m.CollectAndEmitFunc != nil {
		return m.CollectAndEmitFunc(characterId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) Quit(mb *message.Buffer, characterId uint32) (game.Model, error) {
	if m.QuitFunc != nil {
		return m.QuitFunc(mb, characterId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) QuitAndEmit(characterId uint32) (game.Model, error) {
	if m.QuitAndEmitFunc != nil {
		return m.QuitAndEmitFunc(characterId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) Dispose(mb *message.Buffer, characterId uint32) (game.Model, error) {
	if m.DisposeFunc != nil {
		return m.DisposeFunc(mb, characterId)
	}
	return game.Model{}, nil
}

func (m *ProcessorMock) DisposeAndEmit(characterId uint32) (game.Model, error) {
	if m.DisposeAndEmitFunc != nil {
		return m.DisposeAndEmitFunc(characterId)
	}
	return game.Model{}, nil
}
