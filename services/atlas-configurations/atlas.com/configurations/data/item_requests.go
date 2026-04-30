package data

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	equipmentPath = "data/equipment/%d"
)

// ItemRestModel is a minimal rest model used only for existence-checking.
// It mirrors the atlas-data equipment resource shape (resource type "statistics").
type ItemRestModel struct {
	Id uint32 `json:"-"`
}

func (i ItemRestModel) GetName() string {
	return "statistics"
}

func (i ItemRestModel) GetID() string {
	return fmt.Sprint(i.Id)
}

func (i *ItemRestModel) SetID(id string) error {
	var x uint32
	_, err := fmt.Sscan(id, &x)
	if err != nil {
		return err
	}
	i.Id = x
	return nil
}

// requestEquipmentById hits GET /data/equipment/{id} which returns equip statistics.
// A 404 from atlas-data means the template does not exist as equip data.
func requestEquipmentById(id uint32) requests.Request[ItemRestModel] {
	url := fmt.Sprintf("%s"+equipmentPath, getDataBaseRequest(), id)
	return requests.GetRequest[ItemRestModel](url)
}
