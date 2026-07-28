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

// GetID and SetID implement api2go/jsonapi's (Un)MarshalIdentifier
// interfaces. Pre-existing gap: without SetID, jsonapi.Unmarshal rejects
// every element of the response ("target must implement UnmarshalIdentifier
// interface"), which GetBuffsByCharacter's error handling silently
// swallowed into an empty slice — HasActiveBuff/GetBuffsByCharacter have
// likely never returned real data. Surfaced by the task-117 drain-test
// conversion; fixed here since it directly blocks verifying that
// conversion.
func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
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
