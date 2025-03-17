package monster

import (
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
)

type RestModel struct {
	Id                 string            `json:"-"`
	Name               string            `json:"name"`
	HP                 uint32            `json:"hp"`
	MP                 uint32            `json:"mp"`
	Experience         uint32            `json:"experience"`
	Level              uint32            `json:"level"`
	WeaponAttack       uint32            `json:"weapon_attack"`
	WeaponDefense      uint32            `json:"weapon_defense"`
	MagicAttack        uint32            `json:"magic_attack"`
	MagicDefense       uint32            `json:"magic_defense"`
	Friendly           bool              `json:"friendly"`
	RemoveAfter        uint32            `json:"remove_after"`
	Boss               bool              `json:"boss"`
	ExplosiveReward    bool              `json:"explosive_reward"`
	FFALoot            bool              `json:"ffa_loot"`
	Undead             bool              `json:"undead"`
	BuffToGive         uint32            `json:"buff_to_give"`
	CP                 uint32            `json:"cp"`
	RemoveOnMiss       bool              `json:"remove_on_miss"`
	Changeable         bool              `json:"changeable"`
	AnimationTimes     map[string]uint32 `json:"animation_times"`
	Resistances        map[string]string `json:"resistances"`
	LoseItems          []loseItem        `json:"lose_items"`
	Skills             []skill           `json:"skills"`
	Revives            []uint32          `json:"revives"`
	TagColor           byte              `json:"tag_color"`
	TagBackgroundColor byte              `json:"tag_background_color"`
	FixedStance        uint32            `json:"fixed_stance"`
	FirstAttack        bool              `json:"first_attack"`
	Banish             banish            `json:"banish"`
	DropPeriod         uint32            `json:"drop_period"`
	SelfDestruction    selfDestruction   `json:"self_destruction"`
	CoolDamage         coolDamage        `json:"cool_damage"`
}

func (r RestModel) GetName() string {
	return "monsters"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

func Transform(m Model) (RestModel, error) {
	lis, err := model.SliceMap(TransformLoseItem)(model.FixedProvider(m.LoseItems))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}
	ss, err := model.SliceMap(TransformSkill)(model.FixedProvider(m.Skills))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	var b = banish{}
	if m.Banish != nil {
		b, err = model.Map(TransformBanish)(model.FixedProvider(*m.Banish))()
		if err != nil {
			return RestModel{}, err
		}
	}
	var sd = selfDestruction{}
	if m.SelfDestruction != nil {
		sd, err = model.Map(TransformSelfDestruction)(model.FixedProvider(*m.SelfDestruction))()
		if err != nil {
			return RestModel{}, err
		}
	}
	var cd = coolDamage{}
	if m.CoolDamage != nil {
		cd, err = model.Map(TransformCoolDamage)(model.FixedProvider(*m.CoolDamage))()
		if err != nil {
			return RestModel{}, err
		}
	}

	return RestModel{
		Id:                 strconv.Itoa(int(m.Id)),
		Name:               m.Name,
		HP:                 m.HP,
		MP:                 m.MP,
		Experience:         m.Experience,
		Level:              m.Level,
		WeaponAttack:       m.WeaponAttack,
		WeaponDefense:      m.WeaponDefense,
		MagicAttack:        m.MagicAttack,
		MagicDefense:       m.MagicDefense,
		Friendly:           m.Friendly,
		RemoveAfter:        m.RemoveAfter,
		Boss:               m.Boss,
		ExplosiveReward:    m.ExplosiveReward,
		FFALoot:            m.FFALoot,
		Undead:             m.Undead,
		BuffToGive:         m.BuffToGive,
		CP:                 m.CP,
		RemoveOnMiss:       m.RemoveOnMiss,
		Changeable:         m.Changeable,
		AnimationTimes:     m.AnimationTimes,
		Resistances:        m.Resistances,
		LoseItems:          lis,
		Skills:             ss,
		Revives:            m.Revives,
		TagColor:           m.TagColor,
		TagBackgroundColor: m.TagBackgroundColor,
		FixedStance:        m.FixedStance,
		FirstAttack:        m.FirstAttack,
		Banish:             b,
		DropPeriod:         m.DropPeriod,
		SelfDestruction:    sd,
		CoolDamage:         cd,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	lis, err := model.SliceMap(ExtractLoseItem)(model.FixedProvider(rm.LoseItems))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	ss, err := model.SliceMap(ExtractSkill)(model.FixedProvider(rm.Skills))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	b, err := model.Map(ExtractBanish)(model.FixedProvider(rm.Banish))()
	if err != nil {
		return Model{}, err
	}

	sd, err := model.Map(ExtractSelfDestruction)(model.FixedProvider(rm.SelfDestruction))()
	if err != nil {
		return Model{}, err
	}

	cd, err := model.Map(ExtractCoolDamage)(model.FixedProvider(rm.CoolDamage))()
	if err != nil {
		return Model{}, err
	}

	id, err := strconv.Atoi(rm.Id)
	if err != nil {
		return Model{}, err
	}

	return Model{
		Id:                 uint32(id),
		Name:               rm.Name,
		HP:                 rm.HP,
		MP:                 rm.MP,
		Experience:         rm.Experience,
		Level:              rm.Level,
		WeaponAttack:       rm.WeaponAttack,
		WeaponDefense:      rm.WeaponDefense,
		MagicAttack:        rm.MagicAttack,
		MagicDefense:       rm.MagicDefense,
		Friendly:           rm.Friendly,
		RemoveAfter:        rm.RemoveAfter,
		Boss:               rm.Boss,
		ExplosiveReward:    rm.ExplosiveReward,
		FFALoot:            rm.FFALoot,
		Undead:             rm.Undead,
		BuffToGive:         rm.BuffToGive,
		CP:                 rm.CP,
		RemoveOnMiss:       rm.RemoveOnMiss,
		Changeable:         rm.Changeable,
		AnimationTimes:     rm.AnimationTimes,
		Resistances:        rm.Resistances,
		LoseItems:          lis,
		Skills:             ss,
		Revives:            rm.Revives,
		TagColor:           rm.TagColor,
		TagBackgroundColor: rm.TagBackgroundColor,
		FixedStance:        rm.FixedStance,
		FirstAttack:        rm.FirstAttack,
		Banish:             &b,
		DropPeriod:         rm.DropPeriod,
		SelfDestruction:    &sd,
		CoolDamage:         &cd,
	}, nil
}

type loseItem struct {
	Id     uint32 `json:"id"`
	Chance byte   `json:"chance"`
	X      byte   `json:"x"`
}

func TransformLoseItem(m LoseItem) (loseItem, error) {
	return loseItem{
		Id:     m.ItemId,
		Chance: m.Chance,
		X:      m.X,
	}, nil
}

func ExtractLoseItem(rm loseItem) (LoseItem, error) {
	return LoseItem{
		ItemId: rm.Id,
		Chance: rm.Chance,
		X:      rm.X,
	}, nil
}

type skill struct {
	Id    uint32 `json:"id"`
	Level uint32 `json:"level"`
}

func TransformSkill(m Skill) (skill, error) {
	return skill{
		Id:    m.Id,
		Level: m.Level,
	}, nil
}

func ExtractSkill(rm skill) (Skill, error) {
	return Skill{
		Id:    rm.Id,
		Level: rm.Level,
	}, nil
}

type banish struct {
	Message    string `json:"message"`
	MapId      uint32 `json:"map_id"`
	PortalName string `json:"portal_name"`
}

func TransformBanish(m Banish) (banish, error) {
	return banish{
		Message:    m.Message,
		MapId:      m.MapId,
		PortalName: m.PortalName,
	}, nil
}

func ExtractBanish(rm banish) (Banish, error) {
	return Banish{
		Message:    rm.Message,
		MapId:      rm.MapId,
		PortalName: rm.PortalName,
	}, nil
}

type selfDestruction struct {
	Action      byte  `json:"action"`
	RemoveAfter int32 `json:"remove_after"`
	HP          int32 `json:"hp"`
}

func TransformSelfDestruction(m SelfDestruction) (selfDestruction, error) {
	return selfDestruction{
		Action:      m.Action,
		RemoveAfter: m.RemoveAfter,
		HP:          m.HP,
	}, nil
}

func ExtractSelfDestruction(rm selfDestruction) (SelfDestruction, error) {
	return SelfDestruction{
		Action:      rm.Action,
		RemoveAfter: rm.RemoveAfter,
		HP:          rm.HP,
	}, nil
}

type coolDamage struct {
	Damage      uint32 `json:"damage"`
	Probability uint32 `json:"probability"`
}

func TransformCoolDamage(m CoolDamage) (coolDamage, error) {
	return coolDamage{
		Damage:      m.Damage,
		Probability: m.Probability,
	}, nil
}

func ExtractCoolDamage(rm coolDamage) (CoolDamage, error) {
	return CoolDamage{
		Damage:      rm.Damage,
		Probability: rm.Probability,
	}, nil
}

type LoseItemRestModel struct {
	Id     string `json:"-"`
	Chance byte   `json:"chance"`
	X      byte   `json:"x"`
}
