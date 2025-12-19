package tenant

// RestModel is the JSON:API resource for tenants
type RestModel struct {
	Id           string `json:"-"`
	Name         string `json:"name"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return "tenants"
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:           m.Id().String(),
		Name:         m.Name(),
		Region:       m.Region(),
		MajorVersion: m.MajorVersion(),
		MinorVersion: m.MinorVersion(),
	}, nil
}

// Extract converts a RestModel to parameters for creating or updating a Model
func Extract(r RestModel) (Model, error) {
	return NewBuilder().
		SetName(r.Name).
		SetRegion(r.Region).
		SetMajorVersion(r.MajorVersion).
		SetMinorVersion(r.MinorVersion).
		Build(), nil
}
