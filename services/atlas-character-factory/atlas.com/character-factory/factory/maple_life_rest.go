package factory

// MapleLifeCreateRestModel is the player's own submission from the Maple
// Life dialog (design.md §11 A5): the class they picked (by ordinal, not
// job id -- the dialog itself never shows a job id), their look choices, and
// the level (0..10) they chose for the class's SP skill.
type MapleLifeCreateRestModel struct {
	AccountId    uint32 `json:"accountId"`
	WorldId      byte   `json:"worldId"`
	Name         string `json:"name"`
	ClassOrdinal uint32 `json:"classOrdinal"`
	Gender       byte   `json:"gender"`
	Face         uint32 `json:"face"`
	Hair         uint32 `json:"hair"`
	HairColor    uint32 `json:"hairColor"`
	SkinColor    byte   `json:"skinColor"`
	SP           byte   `json:"sp"`
}

// GetName, GetID, and SetID satisfy the jsonapi.MarshalIdentifier /
// jsonapi.UnmarshalIdentifier interfaces so MapleLifeCreateRestModel (defined
// in maple_life.go) can be decoded from a JSON:API request body of type
// "maple-life-create", mirroring PresetCreateRestModel's "preset-create" in
// preset_rest.go.
func (r MapleLifeCreateRestModel) GetName() string     { return "maple-life-create" }
func (r MapleLifeCreateRestModel) GetID() string       { return "" }
func (r *MapleLifeCreateRestModel) SetID(string) error { return nil }
