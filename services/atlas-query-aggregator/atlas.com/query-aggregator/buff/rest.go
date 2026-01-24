package buff

import "time"

// RestModel represents the REST representation of a buff
type RestModel struct {
	Id        string    `json:"-"`
	SourceId  int32     `json:"sourceId"`
	Duration  int32     `json:"duration"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// GetName returns the resource name for JSON:API
func (r RestModel) GetName() string {
	return "buffs"
}

// Extract transforms a RestModel into a domain Model
func Extract(r RestModel) (Model, error) {
	return Model{
		sourceId:  r.SourceId,
		duration:  r.Duration,
		createdAt: r.CreatedAt,
		expiresAt: r.ExpiresAt,
	}, nil
}
