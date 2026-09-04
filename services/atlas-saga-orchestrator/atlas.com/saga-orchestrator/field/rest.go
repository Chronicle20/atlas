package field

// ResetFieldInputRestModel is the body of POST .../reset.
type ResetFieldInputRestModel struct {
	Id         string `json:"-"`
	Difficulty int    `json:"difficulty"`
}

func (r ResetFieldInputRestModel) GetName() string {
	return "maps"
}

func (r ResetFieldInputRestModel) GetID() string {
	return r.Id
}
