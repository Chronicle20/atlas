package mock

import (
	"atlas-query-aggregator/area_info"
	"fmt"
)

// ProcessorImpl is a mock implementation of the area_info.Processor interface.
type ProcessorImpl struct {
	GetByAreaFunc func(characterId uint32, area uint16) (area_info.Model, error)
}

func (m *ProcessorImpl) GetByArea(characterId uint32, area uint16) (area_info.Model, error) {
	if m.GetByAreaFunc != nil {
		return m.GetByAreaFunc(characterId, area)
	}
	return area_info.Model{}, fmt.Errorf("no area info found for character %d area %d", characterId, area)
}
