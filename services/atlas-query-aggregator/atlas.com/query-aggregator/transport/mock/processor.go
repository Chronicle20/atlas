package mock

import (
	"atlas-query-aggregator/transport"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
)

// ProcessorImpl is a mock implementation of the transport.Processor interface
type ProcessorImpl struct {
	GetRouteByStartMapFunc func(mapId _map.Id) (transport.Model, error)
}

// GetRouteByStartMap returns a route by its start map ID
func (m *ProcessorImpl) GetRouteByStartMap(mapId _map.Id) (transport.Model, error) {
	if m.GetRouteByStartMapFunc != nil {
		return m.GetRouteByStartMapFunc(mapId)
	}
	return transport.Model{}, fmt.Errorf("no route found for map %d", mapId)
}
