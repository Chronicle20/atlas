package character

import (
	"atlas-login/equipment"
	"atlas-login/inventory"
	"atlas-login/inventory/compartment"
	"atlas-login/inventory/compartment/asset"
	"atlas-login/pet"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/job"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	id                 uint32
	accountId          uint32
	worldId            world.Id
	name               string
	gender             byte
	skinColor          byte
	face               uint32
	hair               uint32
	level              byte
	jobId              job.Id
	strength           uint16
	dexterity          uint16
	intelligence       uint16
	luck               uint16
	hp                 uint16
	maxHp              uint16
	mp                 uint16
	maxMp              uint16
	hpMpUsed           int
	ap                 uint16
	sp                 string
	experience         uint32
	fame               int16
	gachaponExperience uint32
	mapId              _map.Id
	spawnPoint         uint32
	gm                 int
	meso               uint32
	pets               []pet.Model
	equipment          equipment.Model
	inventory          inventory.Model
}

func (m Model) Gm() bool {
	return m.gm == 1
}

func (m Model) Rank() uint32 {
	return 0
}

func (m Model) RankMove() uint32 {
	return 0
}

func (m Model) JobRank() uint32 {
	return 0
}

func (m Model) JobRankMove() uint32 {
	return 0
}

func (m Model) Gender() byte {
	return m.gender
}

func (m Model) SkinColor() byte {
	return m.skinColor
}

func (m Model) Face() uint32 {
	return m.face
}

func (m Model) Hair() uint32 {
	return m.hair
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) JobId() job.Id {
	return m.jobId
}

func (m Model) Strength() uint16 {
	return m.strength
}

func (m Model) Dexterity() uint16 {
	return m.dexterity
}

func (m Model) Intelligence() uint16 {
	return m.intelligence
}

func (m Model) Luck() uint16 {
	return m.luck
}

func (m Model) Hp() uint16 {
	return m.hp
}

func (m Model) MaxHp() uint16 {
	return m.maxHp
}

func (m Model) Mp() uint16 {
	return m.mp
}

func (m Model) MaxMp() uint16 {
	return m.maxMp
}

func (m Model) Ap() uint16 {
	return m.ap
}

func (m Model) HasSPTable() bool {
	switch m.jobId {
	case 2001:
		return true
	case 2200:
		return true
	case 2210:
		return true
	case 2211:
		return true
	case 2212:
		return true
	case 2213:
		return true
	case 2214:
		return true
	case 2215:
		return true
	case 2216:
		return true
	case 2217:
		return true
	case 2218:
		return true
	default:
		return false
	}
}

func (m Model) Sp() []uint16 {
	s := strings.Split(m.sp, ",")
	var sps = make([]uint16, 0)
	for _, x := range s {
		sp, err := strconv.ParseUint(x, 10, 16)
		if err == nil {
			sps = append(sps, uint16(sp))
		}
	}
	return sps
}

func (m Model) RemainingSp() uint16 {
	return m.Sp()[m.skillBook()]
}

func (m Model) skillBook() uint16 {
	if m.jobId >= 2210 && m.jobId <= 2218 {
		return uint16(m.jobId - 2209)
	}
	return 0
}

func (m Model) Experience() uint32 {
	return m.experience
}

func (m Model) Fame() int16 {
	return m.fame
}

func (m Model) GachaponExperience() uint32 {
	return m.gachaponExperience
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

func (m Model) SpawnPoint() byte {
	return 0
}

func (m Model) Equipment() equipment.Model {
	return m.equipment
}

func (m Model) Pets() []pet.Model {
	return m.pets
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) SetInventory(i inventory.Model) Model {
	eq := equipment.NewModel()
	ec := compartment.NewBuilder(i.Equipable().Id(), m.Id(), i.Equipable().Type(), i.Equipable().Capacity())
	for _, a := range i.Equipable().Assets() {
		if a.Slot() > 0 {
			ec = ec.AddAsset(a)
		} else {
			s := a.Slot()
			cash := s < -100
			if cash {
				s += 100
			}

			es, err := slot.GetSlotByPosition(slot.Position(s))
			if err != nil {
				continue
			}
			v, ok := eq.Get(es.Type)
			if !ok {
				continue
			}

			ea := asset.Clone(a).SetCompartmentId(uuid.Nil).Build()
			if cash {
				v.CashEquipable = &ea
			} else {
				v.Equipable = &ea
			}
			eq.Set(es.Type, v)
		}
	}

	ib := inventory.NewBuilder(m.Id()).
		SetEquipable(ec.Build()).
		SetConsumable(i.Consumable()).
		SetSetup(i.Setup()).
		SetEtc(i.ETC()).
		SetCash(i.Cash())

	return m.ToBuilder().SetInventory(ib.Build()).SetEquipment(eq).Build()
}

// ToBuilder creates a Builder initialized with the Model's values
func (m Model) ToBuilder() *Builder {
	return &Builder{
		id:                 m.id,
		accountId:          m.accountId,
		worldId:            m.worldId,
		name:               m.name,
		gender:             m.gender,
		skinColor:          m.skinColor,
		face:               m.face,
		hair:               m.hair,
		level:              m.level,
		jobId:              m.jobId,
		strength:           m.strength,
		dexterity:          m.dexterity,
		intelligence:       m.intelligence,
		luck:               m.luck,
		hp:                 m.hp,
		maxHp:              m.maxHp,
		mp:                 m.mp,
		maxMp:              m.maxMp,
		hpMpUsed:           m.hpMpUsed,
		ap:                 m.ap,
		sp:                 m.sp,
		experience:         m.experience,
		fame:               m.fame,
		gachaponExperience: m.gachaponExperience,
		mapId:              m.mapId,
		spawnPoint:         m.spawnPoint,
		gm:                 m.gm,
		meso:               m.meso,
		pets:               m.pets,
		equipment:          m.equipment,
		inventory:          m.inventory,
	}
}

// Builder is used to construct a Model instance
type Builder struct {
	id                 uint32
	accountId          uint32
	worldId            world.Id
	name               string
	gender             byte
	skinColor          byte
	face               uint32
	hair               uint32
	level              byte
	jobId              job.Id
	strength           uint16
	dexterity          uint16
	intelligence       uint16
	luck               uint16
	hp                 uint16
	maxHp              uint16
	mp                 uint16
	maxMp              uint16
	hpMpUsed           int
	ap                 uint16
	sp                 string
	experience         uint32
	fame               int16
	gachaponExperience uint32
	mapId              _map.Id
	spawnPoint         uint32
	gm                 int
	meso               uint32
	pets               []pet.Model
	equipment          equipment.Model
	inventory          inventory.Model
}

// NewBuilder creates a new Builder instance
func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) SetId(v uint32) *Builder           { b.id = v; return b }
func (b *Builder) SetAccountId(v uint32) *Builder    { b.accountId = v; return b }
func (b *Builder) SetWorldId(v world.Id) *Builder    { b.worldId = v; return b }
func (b *Builder) SetName(v string) *Builder         { b.name = v; return b }
func (b *Builder) SetGender(v byte) *Builder         { b.gender = v; return b }
func (b *Builder) SetSkinColor(v byte) *Builder      { b.skinColor = v; return b }
func (b *Builder) SetFace(v uint32) *Builder         { b.face = v; return b }
func (b *Builder) SetHair(v uint32) *Builder         { b.hair = v; return b }
func (b *Builder) SetLevel(v byte) *Builder          { b.level = v; return b }
func (b *Builder) SetJobId(v job.Id) *Builder        { b.jobId = v; return b }
func (b *Builder) SetStrength(v uint16) *Builder     { b.strength = v; return b }
func (b *Builder) SetDexterity(v uint16) *Builder    { b.dexterity = v; return b }
func (b *Builder) SetIntelligence(v uint16) *Builder { b.intelligence = v; return b }
func (b *Builder) SetLuck(v uint16) *Builder         { b.luck = v; return b }
func (b *Builder) SetHp(v uint16) *Builder           { b.hp = v; return b }
func (b *Builder) SetMaxHp(v uint16) *Builder        { b.maxHp = v; return b }
func (b *Builder) SetMp(v uint16) *Builder           { b.mp = v; return b }
func (b *Builder) SetMaxMp(v uint16) *Builder        { b.maxMp = v; return b }
func (b *Builder) SetHpMpUsed(v int) *Builder        { b.hpMpUsed = v; return b }
func (b *Builder) SetAp(v uint16) *Builder           { b.ap = v; return b }
func (b *Builder) SetSp(v string) *Builder           { b.sp = v; return b }
func (b *Builder) SetExperience(v uint32) *Builder   { b.experience = v; return b }
func (b *Builder) SetFame(v int16) *Builder          { b.fame = v; return b }
func (b *Builder) SetGachaponExperience(v uint32) *Builder {
	b.gachaponExperience = v
	return b
}
func (b *Builder) SetMapId(v _map.Id) *Builder             { b.mapId = v; return b }
func (b *Builder) SetSpawnPoint(v uint32) *Builder         { b.spawnPoint = v; return b }
func (b *Builder) SetGm(v int) *Builder                    { b.gm = v; return b }
func (b *Builder) SetMeso(v uint32) *Builder               { b.meso = v; return b }
func (b *Builder) SetPets(v []pet.Model) *Builder          { b.pets = v; return b }
func (b *Builder) SetEquipment(v equipment.Model) *Builder { b.equipment = v; return b }
func (b *Builder) SetInventory(v inventory.Model) *Builder { b.inventory = v; return b }

func (b *Builder) Build() Model {
	return Model{
		id:                 b.id,
		accountId:          b.accountId,
		worldId:            b.worldId,
		name:               b.name,
		gender:             b.gender,
		skinColor:          b.skinColor,
		face:               b.face,
		hair:               b.hair,
		level:              b.level,
		jobId:              b.jobId,
		strength:           b.strength,
		dexterity:          b.dexterity,
		intelligence:       b.intelligence,
		luck:               b.luck,
		hp:                 b.hp,
		maxHp:              b.maxHp,
		mp:                 b.mp,
		maxMp:              b.maxMp,
		hpMpUsed:           b.hpMpUsed,
		ap:                 b.ap,
		sp:                 b.sp,
		experience:         b.experience,
		fame:               b.fame,
		gachaponExperience: b.gachaponExperience,
		mapId:              b.mapId,
		spawnPoint:         b.spawnPoint,
		gm:                 b.gm,
		meso:               b.meso,
		pets:               b.pets,
		equipment:          b.equipment,
		inventory:          b.inventory,
	}
}
