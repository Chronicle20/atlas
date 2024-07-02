package character

import (
	"atlas-character/equipment"
	"atlas-character/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
)

type RestModel struct {
	Id                 uint32              `json:"-"`
	AccountId          uint32              `json:"accountId"`
	WorldId            byte                `json:"worldId"`
	Name               string              `json:"name"`
	Level              byte                `json:"level"`
	Experience         uint32              `json:"experience"`
	GachaponExperience uint32              `json:"gachaponExperience"`
	Strength           uint16              `json:"strength"`
	Dexterity          uint16              `json:"dexterity"`
	Intelligence       uint16              `json:"intelligence"`
	Luck               uint16              `json:"luck"`
	Hp                 uint16              `json:"hp"`
	MaxHp              uint16              `json:"maxHp"`
	Mp                 uint16              `json:"mp"`
	MaxMp              uint16              `json:"maxMp"`
	Meso               uint32              `json:"meso"`
	HpMpUsed           int                 `json:"hpMpUsed"`
	JobId              uint16              `json:"jobId"`
	SkinColor          byte                `json:"skinColor"`
	Gender             byte                `json:"gender"`
	Fame               int16               `json:"fame"`
	Hair               uint32              `json:"hair"`
	Face               uint32              `json:"face"`
	Ap                 uint16              `json:"ap"`
	Sp                 string              `json:"sp"`
	MapId              uint32              `json:"mapId"`
	SpawnPoint         uint32              `json:"spawnPoint"`
	Gm                 int                 `json:"gm"`
	X                  int16               `json:"x"`
	Y                  int16               `json:"y"`
	Stance             byte                `json:"stance"`
	Equipment          equipment.RestModel `json:"equipment"`
	Inventory          inventory.RestModel `json:"inventory"`
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

func Transform(m Model) (RestModel, error) {
	td := GetTemporalRegistry().GetById(m.Id())

	eqp, err := equipment.Transform(m.equipment)
	if err != nil {
		return RestModel{}, err
	}
	inv, err := inventory.Transform(m.inventory)
	if err != nil {
		return RestModel{}, err
	}

	rm := RestModel{
		Id:                 m.id,
		AccountId:          m.accountId,
		WorldId:            m.worldId,
		Name:               m.name,
		Level:              m.level,
		Experience:         m.experience,
		GachaponExperience: m.gachaponExperience,
		Strength:           m.strength,
		Dexterity:          m.dexterity,
		Intelligence:       m.intelligence,
		Luck:               m.luck,
		Hp:                 m.hp,
		MaxHp:              m.maxHp,
		Mp:                 m.mp,
		MaxMp:              m.maxMp,
		Meso:               m.meso,
		HpMpUsed:           m.hpMpUsed,
		JobId:              m.jobId,
		SkinColor:          m.skinColor,
		Gender:             m.gender,
		Fame:               m.fame,
		Hair:               m.hair,
		Face:               m.face,
		Ap:                 m.ap,
		Sp:                 m.sp,
		MapId:              m.mapId,
		SpawnPoint:         m.spawnPoint,
		Gm:                 m.gm,
		X:                  td.X(),
		Y:                  td.Y(),
		Stance:             td.Stance(),
		Equipment:          eqp,
		Inventory:          inv,
	}
	return rm, nil
}

func Extract(m RestModel) (Model, error) {
	eqp, err := model.Transform(m.Equipment, equipment.Extract)
	if err != nil {
		return Model{}, err
	}

	inv, err := model.Transform(m.Inventory, inventory.Extract)
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:                 m.Id,
		accountId:          m.AccountId,
		worldId:            m.WorldId,
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
		gm:                 m.Gm,
		equipment:          eqp,
		inventory:          inv,
	}, nil
}
