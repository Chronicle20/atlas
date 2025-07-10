package character

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
	"strings"
)

type RestModel struct {
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
	JobId              uint16 `json:"jobId"`
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
	Reborns            uint32 `json:"reborns"`
	DojoPoints         uint32 `json:"dojoPoints"`
	VanquisherKills    uint32 `json:"vanquisherKills"`
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
	return []jsonapi.Reference{}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	return result
}

func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

func Transform(m Model) (RestModel, error) {
	spStr := strings.Join(func() []string {
		sps := m.Sp()
		result := make([]string, len(sps))
		for i, sp := range sps {
			result[i] = strconv.FormatUint(uint64(sp), 10)
		}
		return result
	}(), ",")

	return RestModel{
		Id:                 m.Id(),
		AccountId:          m.AccountId(),
		WorldId:            byte(m.WorldId()),
		Name:               m.Name(),
		Level:              m.Level(),
		Experience:         m.Experience(),
		GachaponExperience: m.GachaponExperience(),
		Strength:           m.Strength(),
		Dexterity:          m.Dexterity(),
		Intelligence:       m.Intelligence(),
		Luck:               m.Luck(),
		Hp:                 m.Hp(),
		MaxHp:              m.MaxHp(),
		Mp:                 m.Mp(),
		MaxMp:              m.MaxMp(),
		Meso:               m.Meso(),
		HpMpUsed:           m.HpMpUsed(),
		JobId:              m.JobId(),
		SkinColor:          m.SkinColor(),
		Gender:             m.Gender(),
		Fame:               m.Fame(),
		Hair:               m.Hair(),
		Face:               m.Face(),
		Ap:                 m.Ap(),
		Sp:                 spStr,
		MapId:              m.MapId(),
		SpawnPoint:         uint32(m.SpawnPoint()),
		Gm:                 m.GmLevel(),
		X:                  m.X(),
		Y:                  m.Y(),
		Stance:             m.Stance(),
		Reborns:            m.Reborns(),
		DojoPoints:         m.DojoPoints(),
		VanquisherKills:    m.VanquisherKills(),
	}, nil
}

func Extract(m RestModel) (Model, error) {
	return Model{
		id:                 m.Id,
		accountId:          m.AccountId,
		worldId:            world.Id(m.WorldId),
		name:               m.Name,
		level:              m.Level,
		experience:         m.Experience,
		gachaponExperience: m.GachaponExperience,
		strength:           m.Strength,
		dexterity:          m.Dexterity,
		intelligence:       m.Intelligence,
		luck:               m.Luck,
		hp:                 m.Hp,
		mp:                 m.Mp,
		maxHp:              m.MaxHp,
		maxMp:              m.MaxMp,
		meso:               m.Meso,
		hpMpUsed:           m.HpMpUsed,
		jobId:              m.JobId,
		skinColor:          m.SkinColor,
		gender:             m.Gender,
		fame:               m.Fame,
		hair:               m.Hair,
		face:               m.Face,
		ap:                 m.Ap,
		sp:                 m.Sp,
		mapId:              m.MapId,
		spawnPoint:         m.SpawnPoint,
		gm:                 m.Gm,
		reborns:            m.Reborns,
		dojoPoints:         m.DojoPoints,
		vanquisherKills:    m.VanquisherKills,
		x:                  m.X,
		y:                  m.Y,
		stance:             m.Stance,
	}, nil
}
