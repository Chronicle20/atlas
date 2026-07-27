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

// SetID sets the resource ID. Required for jsonapi.Unmarshal
// (UnmarshalIdentifier) -- without it, decoding a response into
// []RestModel always errors ("target must implement UnmarshalIdentifier
// interface"), which was a pre-existing bug surfaced while converting this
// consumer to requests.DrainProvider (task-117): GetPlayerCountInField has
// likely always silently returned 0.
func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

// Extract is an identity transform -- this package only needs the
// character count, not a decoded character id, so requests.DrainProvider is
// parameterized with RestModel on both sides.
func Extract(r RestModel) (RestModel, error) {
	return r, nil
}
