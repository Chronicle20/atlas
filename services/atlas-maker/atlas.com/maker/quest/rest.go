package quest

import "strconv"

// RestModel mirrors atlas-quest's quest-status resource, keeping only the
// attributes this service consumes.
type RestModel struct {
	Id          uint32 `json:"-"`
	CharacterId uint32 `json:"characterId"`
	QuestId     uint32 `json:"questId"`
	State       byte   `json:"state"`
}

func (r RestModel) GetName() string {
	return "quest-status"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = 0
		return nil
	}
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's
// unmarshal even though this client doesn't care about the quest-status
// resource's relationships (libs/atlas-rest gotcha): a target struct must
// implement them or unmarshal errors whenever the upstream response
// includes a relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		characterId: rm.CharacterId,
		questId:     rm.QuestId,
		state:       rm.State,
	}, nil
}
