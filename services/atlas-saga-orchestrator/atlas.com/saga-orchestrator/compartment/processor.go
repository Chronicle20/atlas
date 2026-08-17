package compartment

import (
	asset2 "atlas-saga-orchestrator/kafka/message/asset"
	"atlas-saga-orchestrator/kafka/message/compartment"
	"context"
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// ItemPayload represents an individual item in a transaction
type ItemPayload struct {
	TemplateId uint32    `json:"templateId"`           // TemplateId of the item
	Quantity   uint32    `json:"quantity"`             // Quantity of the item
	Expiration time.Time `json:"expiration,omitempty"` // Expiration time for the item (zero value = no expiration)
}

// CreateAndEquipAssetPayload represents the payload required to create and equip an asset
type CreateAndEquipAssetPayload struct {
	CharacterId     uint32      `json:"characterId"`               // CharacterId associated with the action
	Item            ItemPayload `json:"item"`                      // Item to create and equip
	UseAverageStats bool        `json:"useAverageStats,omitempty"` // UseAverageStats indicates whether average stats should be used when creating the item
}

type Processor interface {
	RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error
	RequestCreateItemWithStats(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time, useAverageStats bool) error
	RequestDestroyItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error
	RequestDestroyAllItems(transactionId uuid.UUID, characterId uint32, templateId uint32) error
	RequestDestroyItemFromSlot(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, quantity uint32) error
	RequestEquipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error
	RequestUnequipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error
	RequestCreateAndEquipAsset(transactionId uuid.UUID, payload CreateAndEquipAssetPayload) error
	RequestAcceptAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) error
	RequestReleaseAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, assetId uint32, quantity uint32) error
	RequestSetOwner(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, owner string) error
	RequestApplyLock(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time) error
	RequestApplyKarma(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, scissorsKarma int32, clear bool) error
	RequestExtendExpiration(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error {
	return p.RequestCreateItemWithStats(transactionId, characterId, templateId, quantity, expiration, false)
}

func (p *ProcessorImpl) RequestCreateItemWithStats(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time, useAverageStats bool) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestCreateAssetCommandProvider(transactionId, characterId, inventoryType, templateId, quantity, expiration, useAverageStats))
}

func (p *ProcessorImpl) RequestDestroyItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}

	// Look up the slot by querying the character's inventory for the templateId
	compartmentModel, err := RequestCompartment(p.l, p.ctx)(characterId, byte(inventoryType))
	if err != nil {
		return errors.New("failed to retrieve inventory compartment: " + err.Error())
	}

	// Find the first asset with matching templateId. NOTE: first, not all —
	// removeAll then saturates the quantity within THIS slot only. See
	// RequestDestroyAllItems for the every-instance variant.
	slot := int16(-1)
	for _, asset := range compartmentModel.Assets {
		if asset.TemplateId == templateId {
			slot = asset.Slot
			break
		}
	}

	if slot == -1 {
		return errors.New("item not found in inventory")
	}

	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestDestroyAssetCommandProvider(transactionId, characterId, inventoryType, slot, quantity, removeAll))
}

// RequestDestroyAllItems destroys every asset matching templateId, across every
// slot of the owning compartment — the thing RequestDestroyItem's
// first-match-then-removeAll cannot express.
//
// Holding no matching asset is NOT an error here: "destroy all of X" over an
// empty set is a satisfied request, not a failed one. That differs deliberately
// from RequestDestroyItem, whose caller named a single item it expected to find.
func (p *ProcessorImpl) RequestDestroyAllItems(transactionId uuid.UUID, characterId uint32, templateId uint32) error {
	inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
	if !ok {
		return errors.New("invalid templateId")
	}

	compartmentModel, err := RequestCompartment(p.l, p.ctx)(characterId, byte(inventoryType))
	if err != nil {
		return errors.New("failed to retrieve inventory compartment: " + err.Error())
	}

	slots := make([]int16, 0, len(compartmentModel.Assets))
	for _, asset := range compartmentModel.Assets {
		if asset.TemplateId == templateId {
			slots = append(slots, asset.Slot)
		}
	}

	if len(slots) == 0 {
		p.l.Debugf("Character [%d] holds no asset [%d] to destroy; nothing to do.", characterId, templateId)
		return nil
	}

	p.l.Debugf("Destroying [%d] instance(s) of asset [%d] for character [%d].", len(slots), templateId, characterId)
	for _, slot := range slots {
		// removeAll per slot: each slot's whole stack goes, and the loop covers
		// every slot the template occupies.
		err = producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestDestroyAssetCommandProvider(transactionId, characterId, inventoryType, slot, 0, true))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessorImpl) RequestDestroyItemFromSlot(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, quantity uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestDestroyAssetCommandProvider(transactionId, characterId, inventory.Type(inventoryType), slot, quantity, false))
}

func (p *ProcessorImpl) RequestEquipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestEquipAssetCommandProvider(transactionId, characterId, inventoryType, source, destination))
}

func (p *ProcessorImpl) RequestUnequipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestUnequipAssetCommandProvider(transactionId, characterId, inventoryType, source, destination))
}

func (p *ProcessorImpl) RequestCreateAndEquipAsset(transactionId uuid.UUID, payload CreateAndEquipAssetPayload) error {
	return p.RequestCreateItemWithStats(transactionId, payload.CharacterId, payload.Item.TemplateId, payload.Item.Quantity, payload.Item.Expiration, payload.UseAverageStats)
}

func (p *ProcessorImpl) RequestAcceptAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, assetData asset2.AssetData) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestAcceptAssetCommandProvider(transactionId, characterId, inventoryType, templateId, assetData))
}

func (p *ProcessorImpl) RequestReleaseAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, assetId uint32, quantity uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestReleaseAssetCommandProvider(transactionId, characterId, inventoryType, assetId, quantity))
}

func (p *ProcessorImpl) RequestSetOwner(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, owner string) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestSetOwnerCommandProvider(transactionId, characterId, inventoryType, slot, owner))
}

func (p *ProcessorImpl) RequestApplyLock(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestApplyLockCommandProvider(transactionId, characterId, inventoryType, slot, expiration))
}

func (p *ProcessorImpl) RequestApplyKarma(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, scissorsKarma int32, clear bool) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestApplyKarmaCommandProvider(transactionId, characterId, inventoryType, slot, scissorsKarma, clear))
}

func (p *ProcessorImpl) RequestExtendExpiration(transactionId uuid.UUID, characterId uint32, inventoryType byte, slot int16, expiration time.Time, extenderTemplateId uint32) error {
	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvCommandTopic)(RequestExtendExpirationCommandProvider(transactionId, characterId, inventoryType, slot, expiration, extenderTemplateId))
}
