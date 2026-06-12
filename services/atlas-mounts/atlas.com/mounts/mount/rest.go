package mount

import (
	"time"
)

// RestModel is the JSON:API representation of a character's mount progression.
// It mirrors Model. Mount is keyed by character (one mount per character) but
// the resource id is the mount's own UUID.
type RestModel struct {
	Id                  string     `json:"-"`
	CharacterId         uint32     `json:"characterId"`
	Level               int        `json:"level"`
	Exp                 int        `json:"exp"`
	Tiredness           int        `json:"tiredness"`
	LastTirednessTickAt *time.Time `json:"lastTirednessTickAt"`
}

func (r RestModel) GetName() string {
	return "mounts"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

// Transform maps a Model into its JSON:API RestModel. The nullable
// LastTirednessTickAt is carried through unchanged (nil when the mount has not
// ticked yet).
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:                  m.Id().String(),
		CharacterId:         m.CharacterId(),
		Level:               m.Level(),
		Exp:                 m.Exp(),
		Tiredness:           m.Tiredness(),
		LastTirednessTickAt: m.LastTirednessTickAt(),
	}, nil
}
