package mock

// ProcessorImpl is a mock implementation of the item.Processor interface
type ProcessorImpl struct {
	GetSlotMaxFunc func(itemId uint32) uint32
}

// GetSlotMax returns the maximum stack size for an item
func (m *ProcessorImpl) GetSlotMax(itemId uint32) uint32 {
	if m.GetSlotMaxFunc != nil {
		return m.GetSlotMaxFunc(itemId)
	}
	return 100 // Default stack size
}
