package character

import (
	"atlas-character/equipment"
	"atlas-character/inventory"
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

func TransformAll(models []Model) []RestModel {
	rms := make([]RestModel, 0)
	for _, m := range models {
		rms = append(rms, Transform(m))
	}
	return rms
}

func Transform(model Model) RestModel {
	td := GetTemporalRegistry().GetById(model.Id())

	rm := RestModel{
		Id:                 model.id,
		AccountId:          model.accountId,
		WorldId:            model.worldId,
		Name:               model.name,
		Level:              model.level,
		Experience:         model.experience,
		GachaponExperience: model.gachaponExperience,
		Strength:           model.strength,
		Dexterity:          model.dexterity,
		Intelligence:       model.intelligence,
		Luck:               model.luck,
		Hp:                 model.hp,
		MaxHp:              model.maxHp,
		Mp:                 model.mp,
		MaxMp:              model.maxMp,
		Meso:               model.meso,
		HpMpUsed:           model.hpMpUsed,
		JobId:              model.jobId,
		SkinColor:          model.skinColor,
		Gender:             model.gender,
		Fame:               model.fame,
		Hair:               model.hair,
		Face:               model.face,
		Ap:                 model.ap,
		Sp:                 model.sp,
		MapId:              model.mapId,
		SpawnPoint:         model.spawnPoint,
		Gm:                 model.gm,
		X:                  td.X(),
		Y:                  td.Y(),
		Stance:             td.Stance(),
		Equipment:          equipment.Transform(model.equipment),
		Inventory:          inventory.Transform(model.inventory),
	}
	return rm
}
