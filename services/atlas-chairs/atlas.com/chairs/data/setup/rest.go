package setup

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id         uint32 `json:"-"`
	RecoveryHP uint32 `json:"recoveryHP"`
	RecoveryMP uint32 `json:"recoveryMP"`
}

func (r RestModel) GetName() string {
	return "setups"
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

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		recoveryHP: rm.RecoveryHP,
		recoveryMP: rm.RecoveryMP,
	}, nil
}
