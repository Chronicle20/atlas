package character

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id                 uint32   `json:"-"`
	AccountId          uint32   `json:"accountId"`
	WorldId            world.Id `json:"worldId"`
	Name               string   `json:"name"`
	Level              byte     `json:"level"`
	Experience         uint32   `json:"experience"`
	GachaponExperience uint32   `json:"gachaponExperience"`
	Strength           uint16   `json:"strength"`
	Dexterity          uint16   `json:"dexterity"`
	Intelligence       uint16   `json:"intelligence"`
	Luck               uint16   `json:"luck"`
	Hp                 uint16   `json:"hp"`
	MaxHp              uint16   `json:"maxHp"`
	Mp                 uint16   `json:"mp"`
	MaxMp              uint16   `json:"maxMp"`
	Meso               uint32   `json:"meso"`
	HpMpUsed           int      `json:"hpMpUsed"`
	JobId              job.Id   `json:"jobId"`
	SkinColor          byte     `json:"skinColor"`
	Gender             byte     `json:"gender"`
	Fame               int16    `json:"fame"`
	Hair               uint32   `json:"hair"`
	Face               uint32   `json:"face"`
	Ap                 uint16   `json:"ap"`
	Sp                 string   `json:"sp"`
	MapId              _map.Id  `json:"mapId"`
	SpawnPoint         uint32   `json:"spawnPoint"`
	Gm                 int      `json:"gm"`
	X                  int16    `json:"x"`
	Y                  int16    `json:"y"`
	Stance             byte     `json:"stance"`
}

func (r RestModel) GetName() string {
	return "characters"
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

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	return result
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
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
		id:                 rm.Id,
		accountId:          rm.AccountId,
		worldId:            rm.WorldId,
		name:               rm.Name,
		gender:             rm.Gender,
		skinColor:          rm.SkinColor,
		face:               rm.Face,
		hair:               rm.Hair,
		level:              rm.Level,
		jobId:              rm.JobId,
		strength:           rm.Strength,
		dexterity:          rm.Dexterity,
		intelligence:       rm.Intelligence,
		luck:               rm.Luck,
		hp:                 rm.Hp,
		maxHp:              rm.MaxHp,
		mp:                 rm.Mp,
		maxMp:              rm.MaxMp,
		hpMpUsed:           rm.HpMpUsed,
		ap:                 rm.Ap,
		sp:                 rm.Sp,
		experience:         rm.Experience,
		fame:               rm.Fame,
		gachaponExperience: rm.GachaponExperience,
		mapId:              rm.MapId,
		spawnPoint:         rm.SpawnPoint,
		gm:                 rm.Gm,
		meso:               rm.Meso,
	}, nil
}
