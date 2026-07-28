package mock

import (
	"atlas-channel/battleship"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// ProcessorMock is a test double for battleship.Processor. Nil Func fields
// return zero values, so callers only need to wire the methods they exercise
// (the established pattern in this repo — see server/mock, session/mock).
type ProcessorMock struct {
	InitShipHPFunc func(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error
	IsRidingFunc   func(characterId uint32) (byte, bool)
	DrainFunc      func(f field.Model, characterId uint32, damage int32) battleship.DrainResult
	ClearFunc      func(characterId uint32)
}

var _ battleship.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InitShipHP(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error {
	if m.InitShipHPFunc != nil {
		return m.InitShipHPFunc(characterId, skillLevel, charLevel, ttl)
	}
	return nil
}

func (m *ProcessorMock) IsRiding(characterId uint32) (byte, bool) {
	if m.IsRidingFunc != nil {
		return m.IsRidingFunc(characterId)
	}
	return 0, false
}

func (m *ProcessorMock) Drain(f field.Model, characterId uint32, damage int32) battleship.DrainResult {
	if m.DrainFunc != nil {
		return m.DrainFunc(f, characterId, damage)
	}
	return battleship.DrainResult{}
}

func (m *ProcessorMock) Clear(characterId uint32) {
	if m.ClearFunc != nil {
		m.ClearFunc(characterId)
	}
}
