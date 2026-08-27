package buff

import (
	"atlas-channel/character/buff/stat"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type RestModel struct {
	Id        string           `json:"-"`
	SourceId  int32            `json:"sourceId"`
	Level     byte             `json:"level"`
	Duration  int32            `json:"duration"`
	Changes   []stat.RestModel `json:"changes"`
	CreatedAt time.Time        `json:"createdAt"`
	ExpiresAt time.Time        `json:"expiresAt"`
	NoExpiry  bool             `json:"noExpiry"`
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
	changes, err := model.SliceMap(stat.Transform)(model.FixedProvider(m.changes))()()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		SourceId:  m.sourceId,
		Level:     m.level,
		Duration:  m.duration,
		Changes:   changes,
		CreatedAt: m.createdAt,
		ExpiresAt: m.expiresAt,
		NoExpiry:  m.noExpiry,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	cs, err := model.SliceMap(stat.Extract)(model.FixedProvider(rm.Changes))()()
	if err != nil {
		return Model{}, err
	}

	return Model{
		sourceId:  rm.SourceId,
		level:     rm.Level,
		duration:  rm.Duration,
		changes:   cs,
		createdAt: rm.CreatedAt,
		expiresAt: rm.ExpiresAt,
		noExpiry:  rm.NoExpiry,
	}, nil
}
