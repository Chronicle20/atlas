package mock

// ProcessorImpl is a mock implementation of the map.Processor interface
type ProcessorImpl struct {
	GetPlayerCountInMapFunc func(worldId byte, channelId byte, mapId uint32) (int, error)
}

// GetPlayerCountInMap returns the player count for a map
func (m *ProcessorImpl) GetPlayerCountInMap(worldId byte, channelId byte, mapId uint32) (int, error) {
	if m.GetPlayerCountInMapFunc != nil {
		return m.GetPlayerCountInMapFunc(worldId, channelId, mapId)
	}
	return 0, nil
}
