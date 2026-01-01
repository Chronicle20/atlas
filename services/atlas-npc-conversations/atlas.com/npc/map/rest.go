package _map

const (
	Resource = "characters"
)

// RestModel represents a character ID from the atlas-maps service
type RestModel struct {
	Id string `json:"-"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id
}
