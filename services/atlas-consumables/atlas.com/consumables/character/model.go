package character

import (
	"atlas-consumables/asset"
	"atlas-consumables/compartment"
	"atlas-consumables/equipment"
	"atlas-consumables/inventory"
	"atlas-consumables/pet"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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
	spawnPoint         uint32
	gm                 int
	x                  int16
	y                  int16
	stance             byte
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
	sps := make([]uint16, 0, len(s))
	for _, x := range s {
		// atlas-character serves the table as ", "-separated. ParseUint rejects
		// a leading space, so an untrimmed entry is dropped silently and the
		// table collapses to its first element.
		sp, err := strconv.ParseUint(strings.TrimSpace(x), 10, 16)
		if err == nil {
			sps = append(sps, uint16(sp))
		}
	}
	return sps
}

func (m Model) RemainingSp() uint16 {
	// Bounds-checked: a short table must not panic the encode path.
	sps := m.Sp()
	if b := int(m.skillBook()); b < len(sps) {
		return sps[b]
	}
	return 0
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

			ea := asset.Clone(a).Build()
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

	return Clone(m).SetInventory(ib.Build()).SetEquipment(eq).Build()
}

func (m Model) SetPets(ms []pet.Model) Model {
	return Clone(m).SetPets(ms).Build()
}
