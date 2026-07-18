package character

import (
	"strconv"
)

// RestModel mirrors the subset of atlas-character's wire format the mini-game
// service needs (id, name, hp).
type RestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
	Hp   uint16 `json:"hp"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are defensive no-op stubs so a
// future relationships block on atlas-character's resource cannot break the
// decode (task-037 failure class, see libs/atlas-rest/CLAUDE.md). The
// mini-game service reads only id/name/hp.
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:   rm.Id,
		name: rm.Name,
		hp:   rm.Hp,
	}, nil
}
