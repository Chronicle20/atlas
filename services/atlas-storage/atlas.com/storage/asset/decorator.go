package asset

import (
	"atlas-storage/equipable"
	"atlas-storage/pet"
	"atlas-storage/stackable"
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Decorator provides methods to enrich assets with their reference data
type Decorator struct {
	l          logrus.FieldLogger
	ctx        context.Context
	db         *gorm.DB
	equipableP equipable.Processor
	petP       pet.Processor
}

func NewDecorator(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Decorator {
	return &Decorator{
		l:          l,
		ctx:        ctx,
		db:         db,
		equipableP: equipable.NewProcessor(l, ctx),
		petP:       pet.NewProcessor(l, ctx),
	}
}

// DecorateEquipable enriches an asset with equipable reference data
func (d *Decorator) DecorateEquipable(m Model[any]) (Model[equipable.Model], error) {
	eq, err := d.equipableP.ByIdProvider(m.referenceId)()
	if err != nil {
		return Model[equipable.Model]{}, err
	}

	return NewModelBuilder[equipable.Model]().
		SetId(m.id).
		SetStorageId(m.storageId).
		SetSlot(m.slot).
		SetTemplateId(m.templateId).
		SetExpiration(m.expiration).
		SetReferenceId(m.referenceId).
		SetReferenceType(m.referenceType).
		SetReferenceData(eq).
		MustBuild(), nil
}

// DecoratePet enriches an asset with pet reference data
func (d *Decorator) DecoratePet(m Model[any]) (Model[pet.Model], error) {
	p, err := d.petP.ByIdProvider(m.referenceId)()
	if err != nil {
		return Model[pet.Model]{}, err
	}

	return NewModelBuilder[pet.Model]().
		SetId(m.id).
		SetStorageId(m.storageId).
		SetSlot(m.slot).
		SetTemplateId(m.templateId).
		SetExpiration(m.expiration).
		SetReferenceId(m.referenceId).
		SetReferenceType(m.referenceType).
		SetReferenceData(p).
		MustBuild(), nil
}

// DecorateStackable enriches an asset with stackable reference data from local storage
func (d *Decorator) DecorateStackable(m Model[any]) (Model[stackable.Model], error) {
	s, err := stackable.GetByAssetId(d.l, d.db)(m.id)
	if err != nil {
		return Model[stackable.Model]{}, err
	}

	return NewModelBuilder[stackable.Model]().
		SetId(m.id).
		SetStorageId(m.storageId).
		SetSlot(m.slot).
		SetTemplateId(m.templateId).
		SetExpiration(m.expiration).
		SetReferenceId(m.referenceId).
		SetReferenceType(m.referenceType).
		SetReferenceData(s).
		MustBuild(), nil
}

// DecorateAssets decorates a slice of assets based on their reference types
// Returns the assets with reference data populated
func (d *Decorator) DecorateAssets(assets []Model[any]) ([]Model[any], error) {
	result := make([]Model[any], 0, len(assets))

	// Collect stackable asset IDs for batch query
	var stackableAssetIds []uint32
	stackableIndexMap := make(map[uint32]int)

	for i, a := range assets {
		if a.IsStackable() {
			stackableAssetIds = append(stackableAssetIds, a.id)
			stackableIndexMap[a.id] = i
		}
	}

	// Batch fetch stackables
	var stackableMap map[uint32]stackable.Model
	if len(stackableAssetIds) > 0 {
		stackables, err := stackable.GetByAssetIds(d.l, d.db)(stackableAssetIds)
		if err != nil {
			return nil, err
		}
		stackableMap = make(map[uint32]stackable.Model)
		for _, s := range stackables {
			stackableMap[s.AssetId()] = s
		}
	}

	for _, a := range assets {
		switch a.referenceType {
		case ReferenceTypeEquipable, ReferenceTypeCashEquipable:
			eq, err := d.equipableP.ByIdProvider(a.referenceId)()
			if err != nil {
				d.l.WithError(err).Warnf("Failed to fetch equipable reference data for asset %d", a.id)
				result = append(result, a)
				continue
			}
			decorated := NewModelBuilder[any]().
				SetId(a.id).
				SetStorageId(a.storageId).
				SetSlot(a.slot).
				SetTemplateId(a.templateId).
				SetExpiration(a.expiration).
				SetReferenceId(a.referenceId).
				SetReferenceType(a.referenceType).
				SetReferenceData(eq).
				MustBuild()
			result = append(result, decorated)

		case ReferenceTypePet:
			p, err := d.petP.ByIdProvider(a.referenceId)()
			if err != nil {
				d.l.WithError(err).Warnf("Failed to fetch pet reference data for asset %d", a.id)
				result = append(result, a)
				continue
			}
			decorated := NewModelBuilder[any]().
				SetId(a.id).
				SetStorageId(a.storageId).
				SetSlot(a.slot).
				SetTemplateId(a.templateId).
				SetExpiration(a.expiration).
				SetReferenceId(a.referenceId).
				SetReferenceType(a.referenceType).
				SetReferenceData(p).
				MustBuild()
			result = append(result, decorated)

		case ReferenceTypeConsumable, ReferenceTypeSetup, ReferenceTypeEtc:
			if s, ok := stackableMap[a.id]; ok {
				decorated := NewModelBuilder[any]().
					SetId(a.id).
					SetStorageId(a.storageId).
					SetSlot(a.slot).
					SetTemplateId(a.templateId).
					SetExpiration(a.expiration).
					SetReferenceId(a.referenceId).
					SetReferenceType(a.referenceType).
					SetReferenceData(s).
					MustBuild()
				result = append(result, decorated)
			} else {
				d.l.Warnf("No stackable data found for asset %d", a.id)
				result = append(result, a)
			}

		case ReferenceTypeCash:
			// Cash items use external atlas-cashshop service
			// For now, leave reference data empty until cash service integration
			result = append(result, a)

		default:
			result = append(result, a)
		}
	}

	return result, nil
}
