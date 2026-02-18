package character

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
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
	MapId              _map.Id   `json:"mapId"`
	Instance           uuid.UUID `json:"instance"`
	SpawnPoint         uint32    `json:"spawnPoint"`
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

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Transform(m Model) (RestModel, error) {
	td := GetTemporalRegistry().GetById(m.Id())
	rm := RestModel{
		Id:                 m.Id(),
		AccountId:          m.AccountId(),
		WorldId:            m.WorldId(),
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
		Ap:                 m.AP(),
		Sp:                 m.SPString(),
		MapId:              m.MapId(),
		Instance:           m.Instance(),
		SpawnPoint:         m.SpawnPoint(),
		Gm:                 m.GM(),
		X:                  td.X(),
		Y:                  td.Y(),
		Stance:             td.Stance(),
	}
	return rm, nil
}

func Extract(m RestModel) (Model, error) {
	return NewModelBuilder().
		SetId(m.Id).
		SetAccountId(m.AccountId).
		SetWorldId(m.WorldId).
		SetName(m.Name).
		SetLevel(m.Level).
		SetExperience(m.Experience).
		SetGachaponExperience(m.GachaponExperience).
		SetStrength(m.Strength).
		SetDexterity(m.Dexterity).
		SetIntelligence(m.Intelligence).
		SetLuck(m.Luck).
		SetHp(m.Hp).
		SetMp(m.Mp).
		SetMaxHp(m.MaxHp).
		SetMaxMp(m.MaxMp).
		SetMeso(m.Meso).
		SetHpMpUsed(m.HpMpUsed).
		SetJobId(m.JobId).
		SetSkinColor(m.SkinColor).
		SetGender(m.Gender).
		SetFame(m.Fame).
		SetHair(m.Hair).
		SetFace(m.Face).
		SetAp(m.Ap).
		SetSp(m.Sp).
		SetMapId(m.MapId).
		SetInstance(m.Instance).
		SetSpawnPoint(m.SpawnPoint).
		SetGm(m.Gm).
		Build(), nil
}
