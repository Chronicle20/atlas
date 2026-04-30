package factory

type PresetCreateRestModel struct {
	PresetId  string `json:"presetId"`
	AccountId uint32 `json:"accountId"`
	WorldId   byte   `json:"worldId"`
	Name      string `json:"name"`
}

func (r PresetCreateRestModel) GetName() string      { return "preset-create" }
func (r PresetCreateRestModel) GetID() string        { return "" }
func (r *PresetCreateRestModel) SetID(string) error  { return nil }
