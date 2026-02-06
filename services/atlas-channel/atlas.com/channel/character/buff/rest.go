package buff

import (
	"atlas-channel/character/buff/stat"
	"time"

	"github.com/Chronicle20/atlas-model/model"
)

type RestModel struct {
	Id        string           `json:"-"`
	SourceId  int32            `json:"sourceId"`
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

func Extract(rm RestModel) (Model, error) {
	cs, err := model.SliceMap(stat.Extract)(model.FixedProvider(rm.Changes))()()
	if err != nil {
		return Model{}, err
	}

	return Model{
		sourceId:  rm.SourceId,
		duration:  rm.Duration,
		changes:   cs,
		createdAt: rm.CreatedAt,
		expiresAt: rm.ExpiresAt,
	}, nil
}
