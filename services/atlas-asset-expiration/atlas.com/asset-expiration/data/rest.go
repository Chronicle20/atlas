package data

type EquipmentRestModel struct {
	Id             uint32 `json:"-"`
	TimeLimited    bool   `json:"timeLimited"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}

func (r EquipmentRestModel) GetName() string {
	return "statistics"
}

func (r EquipmentRestModel) GetID() string {
	return ""
}

func (r *EquipmentRestModel) SetID(id string) error {
	return nil
}

type ConsumableRestModel struct {
	Id             uint32 `json:"-"`
	TimeLimited    bool   `json:"timeLimited"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}

func (r ConsumableRestModel) GetName() string {
	return "consumables"
}

func (r ConsumableRestModel) GetID() string {
	return ""
}

func (r *ConsumableRestModel) SetID(id string) error {
	return nil
}

type SetupRestModel struct {
	Id             uint32 `json:"-"`
	TimeLimited    bool   `json:"timeLimited"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}

func (r SetupRestModel) GetName() string {
	return "setups"
}

func (r SetupRestModel) GetID() string {
	return ""
}

func (r *SetupRestModel) SetID(id string) error {
	return nil
}

type EtcRestModel struct {
	Id             uint32 `json:"-"`
	TimeLimited    bool   `json:"timeLimited"`
	ReplaceItemId  uint32 `json:"replaceItemId,omitempty"`
	ReplaceMessage string `json:"replaceMessage,omitempty"`
}

func (r EtcRestModel) GetName() string {
	return "etcs"
}

func (r EtcRestModel) GetID() string {
	return ""
}

func (r *EtcRestModel) SetID(id string) error {
	return nil
}

// ReplaceInfo holds replacement information for an expired item
type ReplaceInfo struct {
	ReplaceItemId  uint32
	ReplaceMessage string
}
