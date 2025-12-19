package character

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/job"
	"github.com/jtumidanski/api2go/jsonapi"
)

type ForeignRestModel struct {
	Id                 uint32 `json:"-"`
	AccountId          uint32 `json:"accountId"`
	WorldId            byte   `json:"worldId"`
	Name               string `json:"name"`
	Level              byte   `json:"level"`
	Experience         uint32 `json:"experience"`
	GachaponExperience uint32 `json:"gachaponExperience"`
	Strength           uint16 `json:"strength"`
	Dexterity          uint16 `json:"dexterity"`
	Intelligence       uint16 `json:"intelligence"`
	Luck               uint16 `json:"luck"`
	Hp                 uint16 `json:"hp"`
	MaxHp              uint16 `json:"maxHp"`
	Mp                 uint16 `json:"mp"`
	MaxMp              uint16 `json:"maxMp"`
	Meso               uint32 `json:"meso"`
	HpMpUsed           int    `json:"hpMpUsed"`
	JobId              job.Id `json:"jobId"`
	SkinColor          byte   `json:"skinColor"`
	Gender             byte   `json:"gender"`
	Fame               int16  `json:"fame"`
	Hair               uint32 `json:"hair"`
	Face               uint32 `json:"face"`
	Ap                 uint16 `json:"ap"`
	Sp                 string `json:"sp"`
	MapId              uint32 `json:"mapId"`
	SpawnPoint         uint32 `json:"spawnPoint"`
	Gm                 int    `json:"gm"`
	X                  int16  `json:"x"`
	Y                  int16  `json:"y"`
	Stance             byte   `json:"stance"`
}

func (r *ForeignRestModel) GetName() string {
	return "characters"
}

func (r ForeignRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *ForeignRestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	r.Id = uint32(id)
	return nil
}

func (r ForeignRestModel) GetReferences() []jsonapi.Reference {
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

func (r ForeignRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	return result
}

func (r ForeignRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	return result
}

func (r *ForeignRestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *ForeignRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

func (r *ForeignRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

func ExtractForeign(rm ForeignRestModel) (ForeignModel, error) {
	return ForeignModel{
		id:      rm.Id,
		worldId: rm.WorldId,
		mapId:   rm.MapId,
		name:    rm.Name,
		level:   rm.Level,
		jobId:   rm.JobId,
		gm:      rm.Gm,
	}, nil
}
