package _map

type RestModel struct {
	Id                string  `json:"-"`
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
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r *RestModel) SetToOneReferenceID(name string, ID string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{}, nil
}
