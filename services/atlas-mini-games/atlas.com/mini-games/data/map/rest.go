package mapdata

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RestModel mirrors the subset of atlas-data's map wire format the mini-game
// service needs (fieldLimit).
type RestModel struct {
	Id         _map.Id `json:"-"`
	FieldLimit uint32  `json:"fieldLimit"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = _map.Id(id)
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are required no-op stubs:
// atlas-data's map resource ALWAYS emits a relationships block
// (portals/reactors/npcs/monsters), and api2go's UnmarshalToManyRelations
// fails to decode the document if the target model does not implement these.
// The mini-game service only needs fieldLimit, so the relationships are
// intentionally discarded (task-037 failure class, see libs/atlas-rest/CLAUDE.md).
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:         rm.Id,
		fieldLimit: rm.FieldLimit,
	}, nil
}
