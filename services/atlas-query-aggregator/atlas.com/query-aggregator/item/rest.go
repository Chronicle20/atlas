package item

// ConsumableRestModel represents a consumable item from the atlas-data service
type ConsumableRestModel struct {
	Id      uint32  `json:"-"`
	SlotMax uint32  `json:"slotMax"`
	Price   uint32  `json:"price"`
}

// GetName returns the resource name
func (r ConsumableRestModel) GetName() string {
	return "consumables"
}

// GetID returns the resource ID
func (r ConsumableRestModel) GetID() string {
	return ""
}

// SetupRestModel represents a setup item from the atlas-data service
type SetupRestModel struct {
	Id      uint32  `json:"-"`
	SlotMax uint32  `json:"slotMax"`
	Price   uint32  `json:"price"`
}

// GetName returns the resource name
func (r SetupRestModel) GetName() string {
	return "setups"
}

// GetID returns the resource ID
func (r SetupRestModel) GetID() string {
	return ""
}

// EtcRestModel represents an etc item from the atlas-data service
type EtcRestModel struct {
	Id      uint32  `json:"-"`
	SlotMax uint32  `json:"slotMax"`
	Price   uint32  `json:"price"`
}

// GetName returns the resource name
func (r EtcRestModel) GetName() string {
	return "etcs"
}

// GetID returns the resource ID
func (r EtcRestModel) GetID() string {
	return ""
}

// EquipableRestModel represents an equipable item from the atlas-data service
type EquipableRestModel struct {
	Id    uint32 `json:"-"`
	Price uint32 `json:"price"`
}

// GetName returns the resource name
func (r EquipableRestModel) GetName() string {
	return "equipables"
}

// GetID returns the resource ID
func (r EquipableRestModel) GetID() string {
	return ""
}
