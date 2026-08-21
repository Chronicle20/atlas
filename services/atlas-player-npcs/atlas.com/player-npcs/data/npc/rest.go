package npc

import "strconv"

// RestModel mirrors the subset of atlas-data's "npcs" resource (see
// services/atlas-data/atlas.com/data/npc/rest.go) that deploy eligibility
// needs. Imitate absent from the payload decodes to its Go zero value,
// false, matching the server's own default.
type RestModel struct {
	Id      uint32 `json:"-"`
	Name    string `json:"name"`
	Imitate bool   `json:"imitate"`
}

func (r RestModel) GetName() string {
	return "npcs"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:      rm.Id,
		name:    rm.Name,
		imitate: rm.Imitate,
	}, nil
}
