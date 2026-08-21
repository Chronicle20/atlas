package ranking

import "strconv"

// RestModel mirrors atlas-rankings' wire shape (see
// services/atlas-rankings/atlas.com/rankings/ranking/rest.go): resource
// type "rankings", id = characterId, rank/jobRank attributes.
type RestModel struct {
	Id      uint32 `json:"-"`
	Rank    uint32 `json:"rank"`
	JobRank uint32 `json:"jobRank"`
}

func (r RestModel) GetName() string {
	return "rankings"
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
		rank:    rm.Rank,
		jobRank: rm.JobRank,
	}, nil
}
