package character

import (
	"atlas-channel/asset"
	"atlas-channel/character/skill"
	"atlas-channel/compartment"
	"atlas-channel/inventory"
	character2 "atlas-channel/kafka/message/character"
	"atlas-channel/kafka/producer"
	"atlas-channel/pet"
	"atlas-channel/quest"
	"context"
	"errors"
	"sort"

	"github.com/Chronicle20/atlas-constants/field"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor interface defines the operations for character processing
type Processor interface {
	GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error)
	InventoryDecorator(m Model) Model
	PetAssetEnrichmentDecorator(m Model) Model
	SkillModelDecorator(m Model) Model
	QuestModelDecorator(m Model) Model
	GetEquipableInSlot(characterId uint32, slot int16) model.Provider[asset.Model]
	GetItemInSlot(characterId uint32, inventoryType inventory2.Type, slot int16) model.Provider[asset.Model]
	ByNameProvider(name string) model.Provider[[]Model]
	GetByName(name string) (Model, error)
	RequestDistributeAp(f field.Model, characterId uint32, updateTime uint32, distributes []DistributePacket) error
	RequestDropMeso(f field.Model, characterId uint32, amount uint32) error
	ChangeHP(f field.Model, characterId uint32, amount int16) error
	ChangeMP(f field.Model, characterId uint32, amount int16) error
	RequestDistributeSp(f field.Model, characterId uint32, updateTime uint32, skillId uint32, amount int8) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	ip  inventory.Processor
	cp  compartment.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		ip:  inventory.NewProcessor(l, ctx),
		cp:  compartment.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) GetById(decorators ...model.Decorator[Model]) func(characterId uint32) (Model, error) {
	return func(characterId uint32) (Model, error) {
		mp := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)
		return model.Map(model.Decorate(decorators))(mp)()
	}
}

func (p *ProcessorImpl) InventoryDecorator(m Model) Model {
	i, err := p.ip.GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetInventory(i)
}

// PetAssetEnrichmentDecorator fetches pets for a character, sets them on the model, and enriches
// pet assets in the cash compartment with live pet data from atlas-pets.
func (p *ProcessorImpl) PetAssetEnrichmentDecorator(m Model) Model {
	// Fetch all pets for this character
	pp := pet.NewProcessor(p.l, p.ctx)
	pets, err := pp.GetByOwner(m.Id())
	if err != nil {
		p.l.WithError(err).Debugf("Unable to fetch pets for character [%d].", m.Id())
		return m
	}

	// Always set pets on the model (sorted by slot)
	if len(pets) > 0 {
		sort.Slice(pets, func(i, j int) bool {
			return pets[i].Slot() < pets[j].Slot()
		})
		m = m.SetPets(pets)
	}

	// Enrich pet assets in the cash compartment if present
	cashComp := m.Inventory().Cash()
	if len(cashComp.Assets()) == 0 {
		return m
	}

	hasPets := false
	for _, a := range cashComp.Assets() {
		if a.IsPet() {
			hasPets = true
			break
		}
	}
	if !hasPets {
		return m
	}

	// Build lookup by pet ID
	petMap := make(map[uint32]pet.Model)
	for _, pm := range pets {
		petMap[pm.Id()] = pm
	}

	// Enrich pet assets
	enrichedAssets := make([]asset.Model, 0, len(cashComp.Assets()))
	for _, a := range cashComp.Assets() {
		if a.IsPet() {
			if pm, ok := petMap[a.PetId()]; ok {
				a = asset.Clone(a).
					SetPetName(pm.Name()).
					SetPetLevel(pm.Level()).
					SetCloseness(pm.Closeness()).
					SetFullness(pm.Fullness()).
					SetPetSlot(pm.Slot()).
					MustBuild()
			}
		}
		enrichedAssets = append(enrichedAssets, a)
	}

	enrichedCash := compartment.CloneModel(cashComp).SetAssets(enrichedAssets).MustBuild()
	enrichedInventory := inventory.CloneModel(m.Inventory()).SetCash(enrichedCash).MustBuild()
	return m.SetInventory(enrichedInventory)
}

func (p *ProcessorImpl) SkillModelDecorator(m Model) Model {
	ms, err := skill.NewProcessor(p.l, p.ctx).GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetSkills(ms)
}

func (p *ProcessorImpl) QuestModelDecorator(m Model) Model {
	ms, err := quest.NewProcessor(p.l, p.ctx).GetByCharacterId(m.Id())
	if err != nil {
		return m
	}
	return m.SetQuests(ms)
}

func (p *ProcessorImpl) GetEquipableInSlot(characterId uint32, slot int16) model.Provider[asset.Model] {
	cm, err := p.cp.GetByType(characterId, inventory2.TypeValueEquip)
	if err != nil {
		return model.ErrorProvider[asset.Model](err)
	}
	if a, ok := cm.FindBySlot(slot); ok {
		return model.FixedProvider(*a)
	}
	return model.ErrorProvider[asset.Model](errors.New("equipable not found"))
}

func (p *ProcessorImpl) GetItemInSlot(characterId uint32, inventoryType inventory2.Type, slot int16) model.Provider[asset.Model] {
	cm, err := p.cp.GetByType(characterId, inventoryType)
	if err != nil {
		return model.ErrorProvider[asset.Model](err)
	}
	if a, ok := cm.FindBySlot(slot); ok {
		return model.FixedProvider(*a)
	}
	return model.ErrorProvider[asset.Model](errors.New("item not found"))
}

func (p *ProcessorImpl) ByNameProvider(name string) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByName(name), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByName(name string) (Model, error) {
	return model.FirstProvider(p.ByNameProvider(name), model.Filters[Model]())()
}

type DistributePacket struct {
	Flag  uint32
	Value uint32
}

func (p *ProcessorImpl) RequestDistributeAp(f field.Model, characterId uint32, _ uint32, distributes []DistributePacket) error {
	var distributions = make([]character2.DistributePair, 0)
	for _, d := range distributes {
		a, err := abilityFromFlag(d.Flag)
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] passed invalid flag when attempting to distribute AP.", characterId)
			return err
		}

		distributions = append(distributions, character2.DistributePair{
			Ability: a,
			Amount:  int8(d.Value),
		})
	}
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(RequestDistributeApCommandProvider(f, characterId, distributions))
}

func abilityFromFlag(flag uint32) (string, error) {
	switch flag {
	case 64:
		return character2.CommandDistributeApAbilityStrength, nil
	case 128:
		return character2.CommandDistributeApAbilityDexterity, nil
	case 256:
		return character2.CommandDistributeApAbilityIntelligence, nil
	case 512:
		return character2.CommandDistributeApAbilityLuck, nil
	case 2048:
		return character2.CommandDistributeApAbilityHp, nil
	case 8192:
		return character2.CommandDistributeApAbilityMp, nil
	}
	return "", errors.New("invalid flag")
}

func (p *ProcessorImpl) RequestDropMeso(f field.Model, characterId uint32, amount uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(RequestDropMesoCommandProvider(f, characterId, amount))
}

func (p *ProcessorImpl) ChangeHP(f field.Model, characterId uint32, amount int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(ChangeHPCommandProvider(f, characterId, amount))
}

func (p *ProcessorImpl) ChangeMP(f field.Model, characterId uint32, amount int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(ChangeMPCommandProvider(f, characterId, amount))
}

func (p *ProcessorImpl) RequestDistributeSp(f field.Model, characterId uint32, _ uint32, skillId uint32, amount int8) error {
	return producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(RequestDistributeSpCommandProvider(f, characterId, skillId, amount))
}
