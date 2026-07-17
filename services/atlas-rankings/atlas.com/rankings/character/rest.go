package character

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is a trimmed read model of atlas-character's characters
// resource — only the attributes the ranking computation needs.
type RestModel struct {
	Id         uint32   `json:"-"`
	AccountId  uint32   `json:"accountId"`
	WorldId    world.Id `json:"worldId"`
	Level      byte     `json:"level"`
	Experience uint32   `json:"experience"`
	JobId      job.Id   `json:"jobId"`
	Gm         int      `json:"gm"`
}

func (r RestModel) GetName() string {
	return "characters"
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

// Relationship stubs — required because atlas-character responses carry a
// relationships block (equipment/inventories) and api2go errors without them.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		id:         r.Id,
		worldId:    r.WorldId,
		jobId:      r.JobId,
		level:      r.Level,
		experience: r.Experience,
		gm:         r.Gm,
	}, nil
}
