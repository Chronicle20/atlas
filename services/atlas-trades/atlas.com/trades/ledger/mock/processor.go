// Package mock holds a hand-rolled ledger.Processor double for tests in other
// packages.
package mock

import (
	"atlas-trades/ledger"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
)

// ProcessorMock implements ledger.Processor with per-method function fields.
// An unset field makes its method a no-op returning zero values.
type ProcessorMock struct {
	RecordFunc           func(m ledger.Model) (ledger.Model, error)
	GetByIdFunc          func(id uuid.UUID) (ledger.Model, error)
	GetByCharacterIdFunc func(characterId character.Id, from time.Time, to time.Time) ([]ledger.Model, error)
}

func (m *ProcessorMock) Record(e ledger.Model) (ledger.Model, error) {
	if m.RecordFunc != nil {
		return m.RecordFunc(e)
	}
	return ledger.Model{}, nil
}

func (m *ProcessorMock) GetById(id uuid.UUID) (ledger.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return ledger.Model{}, nil
}

func (m *ProcessorMock) GetByCharacterId(characterId character.Id, from time.Time, to time.Time) ([]ledger.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId, from, to)
	}
	return nil, nil
}

var _ ledger.Processor = (*ProcessorMock)(nil)
