package character

import (
	"atlas-cashshop/asset"
	"atlas-cashshop/character/compartment"
	"atlas-cashshop/character/equipment"
	"atlas-cashshop/character/inventory"
	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-constants/job"
	"github.com/Chronicle20/atlas-constants/world"
	"strconv"
	"strings"
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
	jobId              uint16
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
	mapId              uint32
	spawnPoint         uint32
	gm                 int
	x                  int16
	y                  int16
	stance             byte
	meso               uint32
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

func (m Model) JobId() uint16 {
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
	switch job.Id(m.jobId) {
	case job.EvanId:
		return true
	case job.EvanStage1Id:
		return true
	case job.EvanStage2Id:
		return true
	case job.EvanStage3Id:
		return true
	case job.EvanStage4Id:
		return true
	case job.EvanStage5Id:
		return true
	case job.EvanStage6Id:
		return true
	case job.EvanStage7Id:
		return true
	case job.EvanStage8Id:
		return true
	case job.EvanStage9Id:
		return true
	case job.EvanStage10Id:
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
		return m.jobId - 2209
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

func (m Model) MapId() uint32 {
	return m.mapId
}

func (m Model) SpawnPoint() byte {
	return 0
}

func (m Model) Equipment() equipment.Model {
	return m.equipment
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) Meso() uint32 {
	return m.meso
}

func (m Model) Inventory() inventory.Model {
	return m.inventory
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) Stance() byte {
	return m.stance
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) SetInventory(i inventory.Model) Model {
	eq := equipment.NewModel()
	ec := compartment.NewBuilder(i.Equipable().Id(), m.Id(), i.Equipable().Type(), i.Equipable().Capacity())
	for _, a := range i.Equipable().Assets() {
		if a.Slot() > 0 {
			ec = ec.AddAsset(a)
		} else {
			cash := false
			s := a.Slot()
			if s < -100 {
				cash = true
				s -= 100
			}

			es, err := slot.GetSlotByPosition(slot.Position(s))
			if err != nil {
				continue
			}
			v, ok := eq.Get(es.Type)
			if !ok {
				continue
			}

			if cash {
				var crd asset.CashEquipableReferenceData
				crd, ok = a.ReferenceData().(asset.CashEquipableReferenceData)
				if ok {
					ea := asset.NewBuilder[asset.CashEquipableReferenceData](a.Id(), a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(crd).
						Build()
					v.CashEquipable = &ea
				}
			} else {
				var erd asset.EquipableReferenceData
				erd, ok = a.ReferenceData().(asset.EquipableReferenceData)
				if ok {
					ea := asset.NewBuilder[asset.EquipableReferenceData](a.Id(), a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(erd).
						Build()
					v.Equipable = &ea
				}
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

	return Clone(m).SetInventory(ib.Build()).SetEquipment(eq).Build()
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
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
		x:                  m.x,
		y:                  m.y,
		stance:             m.stance,
		meso:               m.meso,
		equipment:          m.equipment,
		inventory:          m.inventory,
	}
}

type ModelBuilder struct {
	id                 uint32
	accountId          uint32
	worldId            world.Id
	name               string
	gender             byte
	skinColor          byte
	face               uint32
	hair               uint32
	level              byte
	jobId              uint16
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
	mapId              uint32
	spawnPoint         uint32
	gm                 int
	x                  int16
	y                  int16
	stance             byte
	meso               uint32
	equipment          equipment.Model
	inventory          inventory.Model
}

func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

func (b *ModelBuilder) SetId(v uint32) *ModelBuilder           { b.id = v; return b }
func (b *ModelBuilder) SetAccountId(v uint32) *ModelBuilder    { b.accountId = v; return b }
func (b *ModelBuilder) SetWorldId(v world.Id) *ModelBuilder    { b.worldId = v; return b }
func (b *ModelBuilder) SetName(v string) *ModelBuilder         { b.name = v; return b }
func (b *ModelBuilder) SetGender(v byte) *ModelBuilder         { b.gender = v; return b }
func (b *ModelBuilder) SetSkinColor(v byte) *ModelBuilder      { b.skinColor = v; return b }
func (b *ModelBuilder) SetFace(v uint32) *ModelBuilder         { b.face = v; return b }
func (b *ModelBuilder) SetHair(v uint32) *ModelBuilder         { b.hair = v; return b }
func (b *ModelBuilder) SetLevel(v byte) *ModelBuilder          { b.level = v; return b }
func (b *ModelBuilder) SetJobId(v uint16) *ModelBuilder        { b.jobId = v; return b }
func (b *ModelBuilder) SetStrength(v uint16) *ModelBuilder     { b.strength = v; return b }
func (b *ModelBuilder) SetDexterity(v uint16) *ModelBuilder    { b.dexterity = v; return b }
func (b *ModelBuilder) SetIntelligence(v uint16) *ModelBuilder { b.intelligence = v; return b }
func (b *ModelBuilder) SetLuck(v uint16) *ModelBuilder         { b.luck = v; return b }
func (b *ModelBuilder) SetHp(v uint16) *ModelBuilder           { b.hp = v; return b }
func (b *ModelBuilder) SetMaxHp(v uint16) *ModelBuilder        { b.maxHp = v; return b }
func (b *ModelBuilder) SetMp(v uint16) *ModelBuilder           { b.mp = v; return b }
func (b *ModelBuilder) SetMaxMp(v uint16) *ModelBuilder        { b.maxMp = v; return b }
func (b *ModelBuilder) SetHpMpUsed(v int) *ModelBuilder        { b.hpMpUsed = v; return b }
func (b *ModelBuilder) SetAp(v uint16) *ModelBuilder           { b.ap = v; return b }
func (b *ModelBuilder) SetSp(v string) *ModelBuilder           { b.sp = v; return b }
func (b *ModelBuilder) SetExperience(v uint32) *ModelBuilder   { b.experience = v; return b }
func (b *ModelBuilder) SetFame(v int16) *ModelBuilder          { b.fame = v; return b }
func (b *ModelBuilder) SetGachaponExperience(v uint32) *ModelBuilder {
	b.gachaponExperience = v
	return b
}
func (b *ModelBuilder) SetMapId(v uint32) *ModelBuilder              { b.mapId = v; return b }
func (b *ModelBuilder) SetSpawnPoint(v uint32) *ModelBuilder         { b.spawnPoint = v; return b }
func (b *ModelBuilder) SetGm(v int) *ModelBuilder                    { b.gm = v; return b }
func (b *ModelBuilder) SetMeso(v uint32) *ModelBuilder               { b.meso = v; return b }
func (b *ModelBuilder) SetEquipment(v equipment.Model) *ModelBuilder { b.equipment = v; return b }
func (b *ModelBuilder) SetInventory(v inventory.Model) *ModelBuilder { b.inventory = v; return b }

func (b *ModelBuilder) Build() Model {
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
		x:                  b.x,
		y:                  b.y,
		stance:             b.stance,
		meso:               b.meso,
		equipment:          b.equipment,
		inventory:          b.inventory,
	}
}
