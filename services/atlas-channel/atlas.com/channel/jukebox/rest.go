package jukebox

type RestModel struct {
	Id         string `json:"-"`
	ItemId     uint32 `json:"itemId"`
	PlayerName string `json:"playerName"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "jukebox"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

// SetToOneReferenceID implements the jsonapi.UnmarshalToOneRelations interface.
// The jukebox resource carries no relationships, so this is a no-op; it
// exists so api2go does not error decoding this GET response (EXT-01;
// see libs/atlas-rest/CLAUDE.md).
func (m *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs implements the jsonapi.UnmarshalToManyRelations interface.
// No relationships on this resource; no-op for interface consistency.
func (m *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}
