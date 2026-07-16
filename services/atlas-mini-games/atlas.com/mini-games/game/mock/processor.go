package mock

import (
	"atlas-mini-games/game"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	CreateFunc              func(txId uuid.UUID, f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error
	VisitFunc               func(txId uuid.UUID, f field.Model, characterId uint32, roomId uint32, password string) error
	LeaveFunc               func(txId uuid.UUID, f field.Model, characterId uint32) error
	ChatFunc                func(txId uuid.UUID, f field.Model, characterId uint32, message string) error
	ExpelFunc               func(txId uuid.UUID, f field.Model, characterId uint32) error
	TeardownCharacterFunc   func(characterId uint32) error
	ReadyFunc               func(txId uuid.UUID, f field.Model, characterId uint32) error
	UnreadyFunc             func(txId uuid.UUID, f field.Model, characterId uint32) error
	StartFunc               func(txId uuid.UUID, f field.Model, characterId uint32) error
	MoveStoneFunc           func(txId uuid.UUID, f field.Model, characterId uint32, x uint32, y uint32, stoneType byte) error
	FlipCardFunc            func(txId uuid.UUID, f field.Model, characterId uint32, first bool, cardIndex byte) error
	RequestTieFunc          func(txId uuid.UUID, f field.Model, characterId uint32) error
	AnswerTieFunc           func(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error
	GiveUpFunc              func(txId uuid.UUID, f field.Model, characterId uint32) error
	RequestRetreatFunc      func(txId uuid.UUID, f field.Model, characterId uint32) error
	AnswerRetreatFunc       func(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error
	SkipFunc                func(txId uuid.UUID, f field.Model, characterId uint32) error
	ExitAfterGameFunc       func(txId uuid.UUID, f field.Model, characterId uint32) error
	CancelExitAfterGameFunc func(txId uuid.UUID, f field.Model, characterId uint32) error
	RoomsInFieldFunc        func(f field.Model) []game.Room
}

func (m *ProcessorMock) Create(txId uuid.UUID, f field.Model, characterId uint32, roomType byte, title string, private bool, password string, pieceType byte) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(txId, f, characterId, roomType, title, private, password, pieceType)
	}
	return nil
}

func (m *ProcessorMock) Visit(txId uuid.UUID, f field.Model, characterId uint32, roomId uint32, password string) error {
	if m.VisitFunc != nil {
		return m.VisitFunc(txId, f, characterId, roomId, password)
	}
	return nil
}

func (m *ProcessorMock) Leave(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.LeaveFunc != nil {
		return m.LeaveFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) Chat(txId uuid.UUID, f field.Model, characterId uint32, message string) error {
	if m.ChatFunc != nil {
		return m.ChatFunc(txId, f, characterId, message)
	}
	return nil
}

func (m *ProcessorMock) Expel(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.ExpelFunc != nil {
		return m.ExpelFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) TeardownCharacter(characterId uint32) error {
	if m.TeardownCharacterFunc != nil {
		return m.TeardownCharacterFunc(characterId)
	}
	return nil
}

func (m *ProcessorMock) Ready(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.ReadyFunc != nil {
		return m.ReadyFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) Unready(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.UnreadyFunc != nil {
		return m.UnreadyFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) Start(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.StartFunc != nil {
		return m.StartFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) MoveStone(txId uuid.UUID, f field.Model, characterId uint32, x uint32, y uint32, stoneType byte) error {
	if m.MoveStoneFunc != nil {
		return m.MoveStoneFunc(txId, f, characterId, x, y, stoneType)
	}
	return nil
}

func (m *ProcessorMock) FlipCard(txId uuid.UUID, f field.Model, characterId uint32, first bool, cardIndex byte) error {
	if m.FlipCardFunc != nil {
		return m.FlipCardFunc(txId, f, characterId, first, cardIndex)
	}
	return nil
}

func (m *ProcessorMock) RequestTie(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.RequestTieFunc != nil {
		return m.RequestTieFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) AnswerTie(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error {
	if m.AnswerTieFunc != nil {
		return m.AnswerTieFunc(txId, f, characterId, accept)
	}
	return nil
}

func (m *ProcessorMock) GiveUp(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.GiveUpFunc != nil {
		return m.GiveUpFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) RequestRetreat(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.RequestRetreatFunc != nil {
		return m.RequestRetreatFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) AnswerRetreat(txId uuid.UUID, f field.Model, characterId uint32, accept bool) error {
	if m.AnswerRetreatFunc != nil {
		return m.AnswerRetreatFunc(txId, f, characterId, accept)
	}
	return nil
}

func (m *ProcessorMock) Skip(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.SkipFunc != nil {
		return m.SkipFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) ExitAfterGame(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.ExitAfterGameFunc != nil {
		return m.ExitAfterGameFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) CancelExitAfterGame(txId uuid.UUID, f field.Model, characterId uint32) error {
	if m.CancelExitAfterGameFunc != nil {
		return m.CancelExitAfterGameFunc(txId, f, characterId)
	}
	return nil
}

func (m *ProcessorMock) RoomsInField(f field.Model) []game.Room {
	if m.RoomsInFieldFunc != nil {
		return m.RoomsInFieldFunc(f)
	}
	return nil
}

var _ game.Processor = (*ProcessorMock)(nil)
