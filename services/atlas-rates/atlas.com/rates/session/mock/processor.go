package mock

import (
	"atlas-rates/session"
	"time"
)

type ProcessorMock struct {
	GetSessionsSinceFunc       func(characterId uint32, since time.Time) ([]session.SessionRestModel, error)
	ComputePlaytimeInRangeFunc func(characterId uint32, start, end time.Time) (time.Duration, error)
}

var _ session.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetSessionsSince(characterId uint32, since time.Time) ([]session.SessionRestModel, error) {
	if m.GetSessionsSinceFunc != nil {
		return m.GetSessionsSinceFunc(characterId, since)
	}
	return nil, nil
}

func (m *ProcessorMock) ComputePlaytimeInRange(characterId uint32, start, end time.Time) (time.Duration, error) {
	if m.ComputePlaytimeInRangeFunc != nil {
		return m.ComputePlaytimeInRangeFunc(characterId, start, end)
	}
	return 0, nil
}
