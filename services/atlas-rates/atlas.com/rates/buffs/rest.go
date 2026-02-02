package buffs

import "time"

// RestModel represents a buff from atlas-buffs
type RestModel struct {
	Id        string          `json:"-"`
	SourceId  int32           `json:"sourceId"`
	Duration  int32           `json:"duration"`
	Changes   []StatRestModel `json:"changes"`
	CreatedAt time.Time       `json:"createdAt"`
	ExpiresAt time.Time       `json:"expiresAt"`
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

// StatRestModel represents a stat change from a buff
type StatRestModel struct {
	Id     string `json:"-"`
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func (r StatRestModel) GetName() string {
	return "stats"
}

func (r StatRestModel) GetID() string {
	return r.Id
}

func (r *StatRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
