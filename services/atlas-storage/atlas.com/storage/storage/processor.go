package storage

import (
	"atlas-storage/asset"
	"atlas-storage/data/consumable"
	"atlas-storage/data/etc"
	"atlas-storage/data/setup"
	"atlas-storage/kafka/message"
	"atlas-storage/kafka/producer"
	"atlas-storage/stackable"
	"context"
	"sort"
	atlasProducer "github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	return &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

// GetOrCreateStorage gets or creates a storage for the given world and account
func (p *Processor) GetOrCreateStorage(worldId byte, accountId uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)

	// Try to get existing storage
	s, err := GetByWorldAndAccountId(p.l, p.db, t.Id())(worldId, accountId)
	if err == nil {
		return s, nil
	}

	// Create new storage
	return Create(p.l, p.db, t.Id())(worldId, accountId)
}

// Deposit deposits an item into storage
func (p *Processor) Deposit(worldId byte, accountId uint32, body message.DepositBody) (uint32, error) {
	t := tenant.MustFromContext(p.ctx)

	// Get or create storage
	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		return 0, err
	}

	// Create asset
	refType := asset.ReferenceType(body.ReferenceType)
	a, err := asset.Create(p.l, p.db, t.Id())(
		s.Id(),
		body.Slot,
		body.TemplateId,
		body.Expiration,
		body.ReferenceId,
		refType,
	)
	if err != nil {
		return 0, err
	}

	// If stackable, create stackable data
	if refType == asset.ReferenceTypeConsumable ||
		refType == asset.ReferenceTypeSetup ||
		refType == asset.ReferenceTypeEtc {
		_, err = stackable.Create(p.l, p.db)(
			a.Id(),
			body.ReferenceData.Quantity,
			body.ReferenceData.OwnerId,
			body.ReferenceData.Flag,
		)
		if err != nil {
			// Rollback asset creation
			_ = asset.Delete(p.l, p.db, t.Id())(a.Id())
			return 0, err
		}
	}

	return a.Id(), nil
}

// DepositAndEmit deposits an item and emits an event
func (p *Processor) DepositAndEmit(transactionId uuid.UUID, worldId byte, accountId uint32, body message.DepositBody) (uint32, error) {
	assetId, err := p.Deposit(worldId, accountId, body)
	if err != nil {
		return 0, err
	}

	// Emit deposited event
	_ = p.emitDepositedEvent(transactionId, worldId, accountId, assetId, body)

	return assetId, nil
}

// Withdraw withdraws an item from storage
func (p *Processor) Withdraw(worldId byte, accountId uint32, body message.WithdrawBody) error {
	t := tenant.MustFromContext(p.ctx)

	// Get asset
	a, err := asset.GetById(p.l, p.db, t.Id())(body.AssetId)
	if err != nil {
		return err
	}

	// If stackable and partial quantity, update quantity instead of deleting
	if a.IsStackable() && body.Quantity > 0 {
		s, err := stackable.GetByAssetId(p.l, p.db)(body.AssetId)
		if err != nil {
			return err
		}

		if body.Quantity < s.Quantity() {
			// Partial withdrawal - update quantity
			return stackable.UpdateQuantity(p.l, p.db)(body.AssetId, s.Quantity()-body.Quantity)
		}
	}

	// Full withdrawal - delete stackable data if exists
	if a.IsStackable() {
		_ = stackable.Delete(p.l, p.db)(body.AssetId)
	}

	// Delete asset
	return asset.Delete(p.l, p.db, t.Id())(body.AssetId)
}

// WithdrawAndEmit withdraws an item and emits an event
func (p *Processor) WithdrawAndEmit(transactionId uuid.UUID, worldId byte, accountId uint32, body message.WithdrawBody) error {
	t := tenant.MustFromContext(p.ctx)

	// Get asset info before withdrawal for the event
	a, err := asset.GetById(p.l, p.db, t.Id())(body.AssetId)
	if err != nil {
		return err
	}

	err = p.Withdraw(worldId, accountId, body)
	if err != nil {
		return err
	}

	// Emit withdrawn event
	_ = p.emitWithdrawnEvent(transactionId, worldId, accountId, a, body.Quantity)

	return nil
}

// UpdateMesos updates the mesos in storage
func (p *Processor) UpdateMesos(worldId byte, accountId uint32, body message.UpdateMesosBody) error {
	t := tenant.MustFromContext(p.ctx)

	s, err := GetByWorldAndAccountId(p.l, p.db, t.Id())(worldId, accountId)
	if err != nil {
		return err
	}

	var newMesos uint32
	switch body.Operation {
	case "SET":
		newMesos = body.Mesos
	case "ADD":
		newMesos = s.Mesos() + body.Mesos
	case "SUBTRACT":
		if s.Mesos() < body.Mesos {
			newMesos = 0
		} else {
			newMesos = s.Mesos() - body.Mesos
		}
	default:
		newMesos = body.Mesos
	}

	return UpdateMesos(p.l, p.db, t.Id())(s.Id(), newMesos)
}

// UpdateMesosAndEmit updates mesos and emits an event
func (p *Processor) UpdateMesosAndEmit(transactionId uuid.UUID, worldId byte, accountId uint32, body message.UpdateMesosBody) error {
	t := tenant.MustFromContext(p.ctx)

	s, err := GetByWorldAndAccountId(p.l, p.db, t.Id())(worldId, accountId)
	if err != nil {
		return err
	}

	oldMesos := s.Mesos()

	err = p.UpdateMesos(worldId, accountId, body)
	if err != nil {
		return err
	}

	// Get updated storage for new mesos
	s, _ = GetByWorldAndAccountId(p.l, p.db, t.Id())(worldId, accountId)

	// Emit mesos updated event
	_ = p.emitMesosUpdatedEvent(transactionId, worldId, accountId, oldMesos, s.Mesos())

	return nil
}

// DepositRollback rolls back a deposit operation
func (p *Processor) DepositRollback(worldId byte, accountId uint32, body message.DepositRollbackBody) error {
	t := tenant.MustFromContext(p.ctx)

	// Get asset to check if stackable
	a, err := asset.GetById(p.l, p.db, t.Id())(body.AssetId)
	if err != nil {
		return err
	}

	// Delete stackable data if exists
	if a.IsStackable() {
		_ = stackable.Delete(p.l, p.db)(body.AssetId)
	}

	// Delete asset
	return asset.Delete(p.l, p.db, t.Id())(body.AssetId)
}

func (p *Processor) emitDepositedEvent(transactionId uuid.UUID, worldId byte, accountId uint32, assetId uint32, body message.DepositBody) error {
	event := &message.StatusEvent[message.DepositedEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeDeposited,
		Body: message.DepositedEventBody{
			AssetId:       assetId,
			Slot:          body.Slot,
			TemplateId:    body.TemplateId,
			ReferenceId:   body.ReferenceId,
			ReferenceType: body.ReferenceType,
			Expiration:    body.Expiration,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func (p *Processor) emitWithdrawnEvent(transactionId uuid.UUID, worldId byte, accountId uint32, a asset.Model[any], quantity uint32) error {
	event := &message.StatusEvent[message.WithdrawnEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeWithdrawn,
		Body: message.WithdrawnEventBody{
			AssetId:    a.Id(),
			Slot:       a.Slot(),
			TemplateId: a.TemplateId(),
			Quantity:   quantity,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func (p *Processor) emitMesosUpdatedEvent(transactionId uuid.UUID, worldId byte, accountId uint32, oldMesos uint32, newMesos uint32) error {
	event := &message.StatusEvent[message.MesosUpdatedEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeMesosUpdated,
		Body: message.MesosUpdatedEventBody{
			OldMesos: oldMesos,
			NewMesos: newMesos,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func createMessageProvider[E any](accountId uint32, event *message.StatusEvent[E]) func() ([]kafka.Message, error) {
	key := atlasProducer.CreateKey(int(accountId))
	return atlasProducer.SingleMessageProvider(key, event)
}

// mergeKey is used to group stackable assets for merging
type mergeKey struct {
	templateId uint32
	ownerId    uint32
	flag       uint16
}

// stackableInfo holds information needed for merging
type stackableInfo struct {
	assetId  uint32
	quantity uint32
	slot     int16
}

// MergeAndSort merges stackable items with same templateId/ownerId/flag and sorts by templateId
// Rules:
// - Only merge items with same ownerId AND same flag
// - Skip rechargeable items (cannot merge)
// - Respect slotMax limits from atlas-data
// - Sort by templateId after merge
func (p *Processor) MergeAndSort(worldId byte, accountId uint32) error {
	t := tenant.MustFromContext(p.ctx)

	// Get storage
	s, err := GetByWorldAndAccountId(p.l, p.db, t.Id())(worldId, accountId)
	if err != nil {
		return err
	}

	// Get all assets
	assets, err := asset.GetByStorageId(p.l, p.db, t.Id())(s.Id())
	if err != nil {
		return err
	}

	// Separate stackables and non-stackables
	var nonStackables []asset.Model[any]
	stackableGroups := make(map[mergeKey][]stackableInfo)
	stackableAssets := make(map[uint32]asset.Model[any])

	// Get all stackable data in batch
	var stackableIds []uint32
	for _, a := range assets {
		if a.IsStackable() {
			stackableIds = append(stackableIds, a.Id())
			stackableAssets[a.Id()] = a
		} else {
			nonStackables = append(nonStackables, a)
		}
	}

	if len(stackableIds) == 0 {
		// No stackables to merge, just sort
		return p.sortAssets(t.Id(), assets)
	}

	stackables, err := stackable.GetByAssetIds(p.l, p.db)(stackableIds)
	if err != nil {
		return err
	}

	// Build stackable lookup map
	stackableMap := make(map[uint32]stackable.Model)
	for _, s := range stackables {
		stackableMap[s.AssetId()] = s
	}

	// Group stackables by mergeKey, checking for rechargeable items
	for assetId, a := range stackableAssets {
		s, ok := stackableMap[assetId]
		if !ok {
			continue
		}

		// Check if consumable is rechargeable (cannot merge)
		if a.ReferenceType() == asset.ReferenceTypeConsumable {
			canMerge, err := p.canMergeConsumable(a.TemplateId())
			if err != nil || !canMerge {
				nonStackables = append(nonStackables, a)
				continue
			}
		}

		key := mergeKey{
			templateId: a.TemplateId(),
			ownerId:    s.OwnerId(),
			flag:       s.Flag(),
		}
		stackableGroups[key] = append(stackableGroups[key], stackableInfo{
			assetId:  assetId,
			quantity: s.Quantity(),
			slot:     a.Slot(),
		})
	}

	// Merge each group and update database
	var mergedAssets []asset.Model[any]

	for key, group := range stackableGroups {
		// Get slotMax for this template
		slotMax, err := p.getSlotMax(key.templateId, stackableAssets[group[0].assetId].ReferenceType())
		if err != nil {
			p.l.WithError(err).Warnf("Failed to get slotMax for template %d, using default of 100", key.templateId)
			slotMax = 100
		}
		if slotMax == 0 {
			slotMax = 100 // Default if not specified
		}

		// Calculate total quantity
		var totalQuantity uint32
		for _, item := range group {
			totalQuantity += item.quantity
		}

		// Determine how many stacks we need
		numStacks := (totalQuantity + slotMax - 1) / slotMax
		if numStacks == 0 {
			numStacks = 1
		}

		// Keep first N assets, delete the rest
		assetsToKeep := min(uint32(len(group)), numStacks)

		// Sort group by slot to keep consistent order
		sort.Slice(group, func(i, j int) bool {
			return group[i].slot < group[j].slot
		})

		// Update quantities for kept assets
		remainingQuantity := totalQuantity
		for i := uint32(0); i < assetsToKeep; i++ {
			item := group[i]
			newQuantity := min(remainingQuantity, slotMax)
			remainingQuantity -= newQuantity

			err := stackable.UpdateQuantity(p.l, p.db)(item.assetId, newQuantity)
			if err != nil {
				return err
			}

			mergedAssets = append(mergedAssets, stackableAssets[item.assetId])
		}

		// Delete excess assets
		for i := int(assetsToKeep); i < len(group); i++ {
			item := group[i]
			_ = stackable.Delete(p.l, p.db)(item.assetId)
			err := asset.Delete(p.l, p.db, t.Id())(item.assetId)
			if err != nil {
				return err
			}
		}
	}

	// Combine all assets for sorting
	allAssets := append(nonStackables, mergedAssets...)

	return p.sortAssets(t.Id(), allAssets)
}

// sortAssets sorts assets by templateId and updates their slots
func (p *Processor) sortAssets(tenantId uuid.UUID, assets []asset.Model[any]) error {
	// Sort by templateId
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].TemplateId() < assets[j].TemplateId()
	})

	// Update slots
	for i, a := range assets {
		newSlot := int16(i + 1) // Slots are 1-indexed
		if a.Slot() != newSlot {
			err := asset.UpdateSlot(p.l, p.db, tenantId)(a.Id(), newSlot)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// canMergeConsumable checks if a consumable item can be merged (not rechargeable)
func (p *Processor) canMergeConsumable(templateId uint32) (bool, error) {
	cp := consumable.NewProcessor(p.l, p.ctx)
	c, err := cp.ByIdProvider(templateId)()
	if err != nil {
		return false, err
	}
	return c.CanMerge(), nil
}

// getSlotMax returns the slotMax for a template based on reference type
func (p *Processor) getSlotMax(templateId uint32, refType asset.ReferenceType) (uint32, error) {
	switch refType {
	case asset.ReferenceTypeConsumable:
		cp := consumable.NewProcessor(p.l, p.ctx)
		c, err := cp.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return c.SlotMax(), nil

	case asset.ReferenceTypeSetup:
		sp := setup.NewProcessor(p.l, p.ctx)
		s, err := sp.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return s.SlotMax(), nil

	case asset.ReferenceTypeEtc:
		ep := etc.NewProcessor(p.l, p.ctx)
		e, err := ep.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return e.SlotMax(), nil

	default:
		return 0, nil
	}
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
