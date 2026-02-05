package _map

import (
	"strconv"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id          uint32  `json:"-"`
	ReturnMapId _map.Id `json:"returnMapId"`
}

func (r RestModel) GetName() string {
	return "maps"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
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
