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
