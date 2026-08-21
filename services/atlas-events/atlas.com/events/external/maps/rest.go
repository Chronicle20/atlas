package maps

import "strconv"

// RestModel is the JSON:API character resource returned by the paginated
// worlds/%d/channels/%d/maps/%d/instances/%s/characters/ list. Only the ID is
// needed to identify who is aboard.
type RestModel struct {
	Id string `json:"-"`
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
	return "characters"
}

// Extract converts a RestModel to the character id it identifies.
func Extract(m RestModel) (uint32, error) {
	id, err := strconv.ParseUint(m.Id, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}
