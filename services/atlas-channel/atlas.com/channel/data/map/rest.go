package map_

import (
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id                uint32  `json:"-"`
	Name              string  `json:"name"`
	StreetName        string  `json:"streetName"`
	ReturnMapId       uint32  `json:"returnMapId"`
	MonsterRate       float64 `json:"monsterRate"`
	OnFirstUserEnter  string  `json:"onFirstUserEnter"`
	OnUserEnter       string  `json:"onUserEnter"`
	FieldLimit        uint32  `json:"fieldLimit"`
	MobInterval       uint32  `json:"mobInterval"`
	Seats             uint32  `json:"seats"`
	Clock             bool    `json:"clock"`
	EverLast          bool    `json:"everLast"`
	Town              bool    `json:"town"`
	DecHP             uint32  `json:"decHP"`
	ProtectItem       uint32  `json:"protectItem"`
	ForcedReturnMapId uint32  `json:"forcedReturnMapId"`
	Boat              bool    `json:"boat"`
	TimeLimit         int32   `json:"timeLimit"`
	FieldType         uint32  `json:"fieldType"`
	MobCapacity       uint32  `json:"mobCapacity"`
	Recovery          float64 `json:"recovery"`
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
		clock: rm.Clock,
	}, nil
}
