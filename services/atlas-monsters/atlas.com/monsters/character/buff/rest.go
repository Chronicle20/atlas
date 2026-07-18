package buff

import "time"

// RestModel is a subset of atlas-buffs' "buffs" JSON:API projection —
// only the fields the hidden-set reconciliation sweep needs. Extra
// attributes in the upstream payload (level, duration, changes, createdAt)
// are ignored by JSON unmarshalling, so a subset model is fine.
type RestModel struct {
	Id        string    `json:"-"`
	SourceId  int32     `json:"sourceId"`
	ExpiresAt time.Time `json:"expiresAt"`
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

// SetToOneReferenceID is a no-op required by api2go's interface.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs is a no-op required by api2go's interface.
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return NewModel(rm.SourceId, rm.ExpiresAt), nil
}
