package data

import "github.com/jtumidanski/api2go/jsonapi"

type RestModel struct {
	Id          string `json:"-"`
	ReturnMapId uint32 `json:"returnMapId"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r *RestModel) SetToOneReferenceID(name string, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		returnMapId: rm.ReturnMapId,
	}, nil
}
