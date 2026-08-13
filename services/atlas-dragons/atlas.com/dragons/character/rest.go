package character

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

type RestModel struct {
	Id     uint32 `json:"-"`
	JobId  job.Id `json:"jobId"`
	X      int16  `json:"x"`
	Y      int16  `json:"y"`
	Stance byte   `json:"stance"`
}

func (r RestModel) GetName() string { return "characters" }

func (r RestModel) GetID() string { return strconv.Itoa(int(r.Id)) }

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(m RestModel) (Model, error) {
	return Model{id: m.Id, jobId: m.JobId, x: m.X, y: m.Y, stance: m.Stance}, nil
}
