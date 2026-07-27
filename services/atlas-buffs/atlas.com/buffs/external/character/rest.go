package character

import "strconv"

// RestModel is the trimmed atlas-character projection this service reads:
// current HP and character level per re-evaluation (design D5).
type RestModel struct {
	Id    uint32 `json:"-"`
	Level byte   `json:"level"`
	Hp    uint16 `json:"hp"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
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
