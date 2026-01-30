package data

// FaceRestModel represents the face response from atlas-data service
type FaceRestModel struct {
	Id   string `json:"-"`
	Cash bool   `json:"cash"`
}

// GetName returns the JSON:API type name
func (r FaceRestModel) GetName() string {
	return "faces"
}

// GetID returns the JSON:API resource ID
func (r FaceRestModel) GetID() string {
	return r.Id
}

// SetID sets the JSON:API resource ID
func (r *FaceRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// HairRestModel represents the hair response from atlas-data service
type HairRestModel struct {
	Id   string `json:"-"`
	Cash bool   `json:"cash"`
}

// GetName returns the JSON:API type name
func (r HairRestModel) GetName() string {
	return "hairs"
}

// GetID returns the JSON:API resource ID
func (r HairRestModel) GetID() string {
	return r.Id
}

// SetID sets the JSON:API resource ID
func (r *HairRestModel) SetID(id string) error {
	r.Id = id
	return nil
}
