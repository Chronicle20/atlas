package character

import (
	"atlas-channel/asset"
	"atlas-channel/character/skill"
	"atlas-channel/compartment"
	"atlas-channel/equipment"
	"atlas-channel/inventory"
	"atlas-channel/pet"
	"atlas-channel/quest"
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
	x                  int16
	y                  int16
	stance             byte
	meso               uint32
	pets               []pet.Model
	equipment          equipment.Model
	inventory          inventory.Model
	skills             []skill.Model
	quests             []quest.Model
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

func (m Model) Skills() []skill.Model {
	return m.skills
}

func (m Model) Quests() []quest.Model {
	return m.quests
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

			if cash {
				var crd asset.CashEquipableReferenceData
				crd, ok = a.ReferenceData().(asset.CashEquipableReferenceData)
				if ok {
					ea := asset.NewBuilder[asset.CashEquipableReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(crd).
						MustBuild()
					v.CashEquipable = &ea
				}
			} else {
				var erd asset.EquipableReferenceData
				erd, ok = a.ReferenceData().(asset.EquipableReferenceData)
				if ok {
					ea := asset.NewBuilder[asset.EquipableReferenceData](a.Id(), uuid.Nil, a.TemplateId(), a.ReferenceId(), a.ReferenceType()).
						SetSlot(a.Slot()).
						SetExpiration(a.Expiration()).
						SetReferenceData(erd).
						MustBuild()
					v.Equipable = &ea
				}
			}
			eq.Set(es.Type, v)
		}
	}

	ib := inventory.NewBuilder(m.Id()).
		SetEquipable(ec.MustBuild()).
		SetConsumable(i.Consumable()).
		SetSetup(i.Setup()).
		SetEtc(i.ETC()).
		SetCash(i.Cash())

	return CloneModel(m).SetInventory(ib.MustBuild()).SetEquipment(eq).MustBuild()
}

func (m Model) SetSkills(ms []skill.Model) Model {
	return CloneModel(m).SetSkills(ms).MustBuild()
}

func (m Model) SetPets(ms []pet.Model) Model {
	return CloneModel(m).SetPets(ms).MustBuild()
}

func (m Model) SetQuests(ms []quest.Model) Model {
	return CloneModel(m).SetQuests(ms).MustBuild()
}
