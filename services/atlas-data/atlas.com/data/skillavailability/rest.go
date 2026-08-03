package skillavailability

import "strconv"

// RestModel is one skill identity available (released/playable) at the
// requesting tenant's client version. Id is the version-appropriate wire
// id (skill.Id is uint32) -- NOT the version-blind skill.Identity token --
// so the frontend preset selector can round-trip it straight back into
// whatever skill-id field the tenant's version expects.
type RestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (r RestModel) GetName() string { return "skill-availability" }
func (r RestModel) GetID() string   { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
