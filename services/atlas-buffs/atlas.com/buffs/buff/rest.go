package buff

import (
	"atlas-buffs/buff/stat"
	"time"

	"github.com/Chronicle20/atlas-model/model"
)

type RestModel struct {
	Id        string           `json:"-"`
	SourceId  int32            `json:"sourceId"`
	Level     byte             `json:"level"`
	Duration  int32            `json:"duration"`
	Changes   []stat.RestModel `json:"changes"`
	CreatedAt time.Time        `json:"createdAt"`
	ExpiresAt time.Time        `json:"expiresAt"`
}

func (r RestModel) GetName() string {
	return "buffs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Transform(m Model) (RestModel, error) {
	cs, err := model.SliceMap(stat.Transform)(model.FixedProvider(m.changes))()()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:        m.id.String(),
		SourceId:  m.sourceId,
		Level:     m.level,
		Duration:  m.duration,
		Changes:   cs,
		CreatedAt: m.createdAt,
		ExpiresAt: m.expiresAt,
	}, nil
}
