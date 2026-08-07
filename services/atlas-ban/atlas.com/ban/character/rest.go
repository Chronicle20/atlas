package character

import "strconv"

type RestModel struct {
	Id   uint32 `json:"-"`
	Name string `json:"name"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs are required even though this
// client doesn't use relationship data: api2go's jsonapi.Unmarshal fails the
// entire decode if the response carries a relationships block and the target
// struct doesn't implement these (see libs/atlas-rest/CLAUDE.md). Mirrors
// atlas-fame/character/rest.go, which hits the same characters/{id} resource.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{id: rm.Id, name: rm.Name}, nil
}
