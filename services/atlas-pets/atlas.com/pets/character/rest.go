package character

import (
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id     uint32 `json:"-"`
	X      int16  `json:"x"`
	Y      int16  `json:"y"`
	Stance byte   `json:"stance"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "equipment",
			Name: "equipment",
		},
		{
			Type: "inventories",
			Name: "inventories",
		},
	}
}

func Extract(m RestModel) (Model, error) {
	return Model{
		id:     m.Id,
		x:      m.X,
		y:      m.Y,
		stance: m.Stance,
	}, nil
}
