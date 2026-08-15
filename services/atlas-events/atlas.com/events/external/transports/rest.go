package transports

// RestModel mirrors only the fields this service reads from the
// atlas-transports route resource: is it still in_transit, and on which
// voyage. See transport.RestModel in atlas-transports for the full resource.
type RestModel struct {
	Id       string `json:"-"`
	State    string `json:"state"`
	VoyageID string `json:"voyageId,omitempty"`
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "routes"
}

// SetToOneReferenceID and SetToManyReferenceIDs are required by api2go
// (jsonapi.Unmarshal) because the upstream transport.RestModel unconditionally
// declares a to-many "schedule" relationship in GetReferences(), so every
// real response carries a relationships.schedule.data array -- even though
// this client doesn't care about the relationship payload. See
// libs/atlas-rest/CLAUDE.md.
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
