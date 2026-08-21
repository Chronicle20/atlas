package factory

// GetName, GetID, and SetID satisfy the jsonapi.MarshalIdentifier /
// jsonapi.UnmarshalIdentifier interfaces so MapleLifeCreateRestModel (defined
// in maple_life.go) can be decoded from a JSON:API request body of type
// "maple-life-create", mirroring PresetCreateRestModel's "preset-create" in
// preset_rest.go.
func (r MapleLifeCreateRestModel) GetName() string     { return "maple-life-create" }
func (r MapleLifeCreateRestModel) GetID() string       { return "" }
func (r *MapleLifeCreateRestModel) SetID(string) error { return nil }
