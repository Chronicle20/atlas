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
	return NewBuilder(m.Id).SetJobId(m.JobId).SetX(m.X).SetY(m.Y).SetStance(m.Stance).Build()
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:     m.id,
		JobId:  m.jobId,
		X:      m.x,
		Y:      m.y,
		Stance: m.stance,
	}, nil
}
