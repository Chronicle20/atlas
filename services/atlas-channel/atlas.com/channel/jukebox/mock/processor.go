package mock

import (
	"atlas-channel/jukebox"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	GetActiveFunc func(f field.Model) (jukebox.RestModel, error)
}

var _ jukebox.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetActive(f field.Model) (jukebox.RestModel, error) {
	if m.GetActiveFunc != nil {
		return m.GetActiveFunc(f)
	}
	return jukebox.RestModel{}, nil
}
