package item

import "strconv"

// RestModel mirrors atlas-data's item-string search result (item/string_rest.go
// StringRestModel / string_resource.go StringSearchResultRestModel): the JSON:API
// resource id is the item template id (string) and `name` is the item name. Only
// the id and name are consumed by the marketplace name search.
type RestModel struct {
	Id   string `json:"-"`
	Name string `json:"name"`
}

// GetName returns the JSON:API resource type. It MUST match atlas-data's
// StringRestModel.GetName() ("item-strings") so api2go unmarshal accepts the
// response.
func (r RestModel) GetName() string {
	return "item-strings"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's unmarshal
// even though the item-string resource carries no relationships (see
// libs/atlas-rest gotcha): a target struct must implement them or unmarshal errors.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Extract parses the JSON:API id (the item template id as a decimal string) into
// the model's uint32 template id.
func Extract(rm RestModel) (Model, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return Model{}, err
	}
	return Model{
		itemId: uint32(id),
		name:   rm.Name,
	}, nil
}
