package mock

import (
	"github.com/Chronicle20/atlas-constants/field"
)

// ProcessorImpl is a mock implementation of the map.Processor interface
type ProcessorImpl struct {
	GetPlayerCountInMapFunc func(f field.Model) (int, error)
}

// GetPlayerCountInMap returns the player count for a map
func (m *ProcessorImpl) GetPlayerCountInMap(f field.Model) (int, error) {
	if m.GetPlayerCountInMapFunc != nil {
		return m.GetPlayerCountInMapFunc(f)
	}
	return 0, nil
}
