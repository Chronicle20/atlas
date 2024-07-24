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

func Transform(m Model) (RestModel, error) {
	lis, err := model.SliceMap(model.FixedProvider(m.loseItems), TransformLoseItem)()
	if err != nil {
		return RestModel{}, err
	}
	ss, err := model.SliceMap(model.FixedProvider(m.skills), TransformSkill)()
	if err != nil {
		return RestModel{}, err
	}

	var b banish = banish{}
	if m.banish != nil {
		b, err = model.Map(model.FixedProvider(*m.banish), TransformBanish)()
		if err != nil {
			return RestModel{}, err
		}
	}
	var sd selfDestruction = selfDestruction{}
	if m.selfDestruction != nil {
		sd, err = model.Map(model.FixedProvider(*m.selfDestruction), TransformSelfDestruction)()
		if err != nil {
			return RestModel{}, err
		}
	}
	var cd coolDamage = coolDamage{}
	if m.coolDamage != nil {
		cd, err = model.Map(model.FixedProvider(*m.coolDamage), TransformCoolDamage)()
		if err != nil {
			return RestModel{}, err
		}
	}

	return RestModel{
		Id:                 strconv.Itoa(int(m.id)),
		Name:               m.name,
		HP:                 m.hp,
		MP:                 m.mp,
		Experience:         m.experience,
		Level:              m.level,
		WeaponAttack:       m.weaponAttack,
		WeaponDefense:      m.weaponDefense,
		MagicAttack:        m.magicAttack,
		MagicDefense:       m.magicDefense,
		Friendly:           m.friendly,
		RemoveAfter:        m.removeAfter,
		Boss:               m.boss,
		ExplosiveReward:    m.explosiveReward,
		FFALoot:            m.ffaLoot,
		Undead:             m.undead,
		BuffToGive:         m.buffToGive,
		CP:                 m.cp,
		RemoveOnMiss:       m.removeOnMiss,
		Changeable:         m.changeable,
		AnimationTimes:     m.animationTimes,
		Resistances:        m.resistances,
		LoseItems:          lis,
		Skills:             ss,
		Revives:            m.revives,
		TagColor:           m.tagColor,
		TagBackgroundColor: m.tagBackgroundColor,
		FixedStance:        m.fixedStance,
		FirstAttack:        m.firstAttack,
		Banish:             b,
		DropPeriod:         m.dropPeriod,
		SelfDestruction:    sd,
		CoolDamage:         cd,
	}, nil
}

type loseItem struct {
	Id     uint32 `json:"id"`
	Chance byte   `json:"chance"`
	X      byte   `json:"x"`
}

func TransformLoseItem(m LoseItem) (loseItem, error) {
	return loseItem{
		Id:     m.itemId,
		Chance: m.chance,
		X:      m.x,
	}, nil
}

type skill struct {
	Id    uint32 `json:"id"`
	Level uint32 `json:"level"`
}

func TransformSkill(m Skill) (skill, error) {
	return skill{
		Id:    m.id,
		Level: m.level,
	}, nil
}

type banish struct {
	Message    string `json:"message"`
	MapId      uint32 `json:"map_id"`
	PortalName string `json:"portal_name"`
}

func TransformBanish(m Banish) (banish, error) {
	return banish{
		Message:    m.message,
		MapId:      m.mapId,
		PortalName: m.portalName,
	}, nil
}

type selfDestruction struct {
	Action      byte  `json:"action"`
	RemoveAfter int32 `json:"remove_after"`
	HP          int32 `json:"hp"`
}

func TransformSelfDestruction(m SelfDestruction) (selfDestruction, error) {
	return selfDestruction{
		Action:      m.action,
		RemoveAfter: m.removeAfter,
		HP:          m.hp,
	}, nil
}

type coolDamage struct {
	Damage      uint32 `json:"damage"`
	Probability uint32 `json:"probability"`
}

func TransformCoolDamage(m CoolDamage) (coolDamage, error) {
	return coolDamage{
		Damage:      m.damage,
		Probability: m.probability,
	}, nil
}

type LoseItemRestModel struct {
	Id     string `json:"-"`
	Chance byte   `json:"chance"`
	X      byte   `json:"x"`
}
