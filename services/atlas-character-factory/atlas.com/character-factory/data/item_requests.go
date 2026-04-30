package data

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const equipmentPath = "data/equipment/%d"

type EquipmentRestModel struct {
	Id uint32 `json:"-"`
}

func (e EquipmentRestModel) GetName() string { return "statistics" }
func (e EquipmentRestModel) GetID() string   { return fmt.Sprint(e.Id) }
func (e *EquipmentRestModel) SetID(id string) error {
	var x uint64
	if _, err := fmt.Sscan(id, &x); err != nil {
		return err
	}
	e.Id = uint32(x)
	return nil
}

func requestEquipmentById(id uint32) requests.Request[EquipmentRestModel] {
	return requests.GetRequest[EquipmentRestModel](fmt.Sprintf("%s"+equipmentPath, getDataBaseRequest(), id))
}
