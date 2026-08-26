package character

import (
	"strconv"
)

// RestModel is the minimal projection of the atlas-character JSON:API
// resource needed by atlas-reactors. atlas-character exposes many more
// attributes; only position is consumed by the touch-proximity check.
type RestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

// GetName returns the JSON:API resource type. Must match atlas-character.
func (r RestModel) GetName() string {
	return "characters"
}

// GetID returns the JSON:API resource id.
func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

// SetID parses the JSON:API resource id back into the model.
func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go
// (jsonapi.Unmarshal) if the upstream response ever carries a
// `relationships` block, even when this client doesn't care about the
// relationship payload. See libs/atlas-rest/CLAUDE.md.
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
