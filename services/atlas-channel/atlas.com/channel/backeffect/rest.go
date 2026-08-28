package backeffect

import (
	beconst "github.com/Chronicle20/atlas/libs/atlas-constants/backeffect"
)

// RestModel is the channel's projection of the atlas-maps back-effect
// resource. Field names and GetName() below must match
// services/atlas-maps/atlas.com/maps/map/backeffect/rest.go exactly.
type RestModel struct {
	Id       string         `json:"-"`
	Effect   beconst.Effect `json:"effect"`
	FieldId  uint32         `json:"fieldId"`
	PageId   uint8          `json:"pageId"`
	Duration uint32         `json:"duration"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "backEffect"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go
// (jsonapi.Unmarshal) if the upstream response ever carries a
// `relationships` block, even when this client doesn't care about the
// relationship payload. See libs/atlas-rest/CLAUDE.md.
func (m *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (m *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func Extract(m RestModel) (RestModel, error) {
	return m, nil
}
