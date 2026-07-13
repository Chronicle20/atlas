package storage

import (
	"atlas-storage/asset"
	"atlas-storage/data/consumable"
	"atlas-storage/data/etc"
	"atlas-storage/data/setup"
	"atlas-storage/kafka/message"
	"atlas-storage/kafka/message/compartment"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"context"
	"errors"
	"sort"

	assetConstants "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetOrCreateStorage(worldId world.Id, accountId uint32) (Model, error)
	GetStorageByWorldAndAccountId(worldId world.Id, accountId uint32) (Model, error)
	CreateStorage(worldId world.Id, accountId uint32) (Model, error)
	Deposit(worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error)
	DepositAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error)
	Withdraw(body message.WithdrawBody) error
	WithdrawAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.WithdrawBody) error
	UpdateMesos(worldId world.Id, accountId uint32, body message.UpdateMesosBody) error
	UpdateMesosAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.UpdateMesosBody) error
	DepositRollback(body message.DepositRollbackBody) error
	Accept(worldId world.Id, accountId uint32, body compartment.AcceptCommandBody) (uint32, int16, error)
	AcceptAndEmit(worldId world.Id, accountId uint32, characterId uint32, body compartment.AcceptCommandBody) error
	Release(body compartment.ReleaseCommandBody) error
	ReleaseAndEmit(worldId world.Id, accountId uint32, characterId uint32, body compartment.ReleaseCommandBody) error
	MergeAndSort(worldId world.Id, accountId uint32) error
	ArrangeAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32) error
	EmitProjectionCreatedEvent(characterId uint32, accountId uint32, ch channel.Model, npcId uint32) error
	ExpireAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId uint32, isCash bool, replaceItemId uint32, replaceMessage string) error
	DeleteByAccountId(accountId uint32) error
	EmitProjectionDestroyedEvent(characterId uint32, accountId uint32, worldId world.Id) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// WithTransaction returns a clone of the processor bound to the transaction handle.
func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) *ProcessorImpl {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: tx}
}

func (p *ProcessorImpl) GetOrCreateStorage(worldId world.Id, accountId uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)

	s, err := GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)
	if err == nil {
		return s, nil
	}

	return Create(p.l, p.db.WithContext(p.ctx), t.Id())(worldId, accountId)
}

func (p *ProcessorImpl) GetStorageByWorldAndAccountId(worldId world.Id, accountId uint32) (Model, error) {
	return GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)
}

func (p *ProcessorImpl) CreateStorage(worldId world.Id, accountId uint32) (Model, error) {
	t := tenant.MustFromContext(p.ctx)

	_, err := GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)
	if err == nil {
		return Model{}, errors.New("storage already exists")
	}

	return Create(p.l, p.db.WithContext(p.ctx), t.Id())(worldId, accountId)
}

func (p *ProcessorImpl) Deposit(worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error) {
	t := tenant.MustFromContext(p.ctx)

	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		return 0, err
	}

	m := asset.NewBuilder(s.Id(), body.TemplateId).
		SetSlot(body.Slot).
		SetExpiration(body.Expiration).
		SetQuantity(body.Quantity).
		SetOwnerId(body.OwnerId).
		SetFlag(body.Flag).
		SetRechargeable(body.Rechargeable).
		SetStrength(body.Strength).
		SetDexterity(body.Dexterity).
		SetIntelligence(body.Intelligence).
		SetLuck(body.Luck).
		SetHp(body.Hp).
		SetMp(body.Mp).
		SetWeaponAttack(body.WeaponAttack).
		SetMagicAttack(body.MagicAttack).
		SetWeaponDefense(body.WeaponDefense).
		SetMagicDefense(body.MagicDefense).
		SetAccuracy(body.Accuracy).
		SetAvoidability(body.Avoidability).
		SetHands(body.Hands).
		SetSpeed(body.Speed).
		SetJump(body.Jump).
		SetSlots(body.Slots).
		SetLevelType(body.LevelType).
		SetLevel(body.Level).
		SetExperience(body.Experience).
		SetHammersApplied(body.HammersApplied).
		SetCashId(body.CashId).
		SetCommodityId(body.CommodityId).
		SetPurchaseBy(body.PurchaseBy).
		SetPetId(body.PetId).
		Build()

	a, err := asset.Create(p.l, p.db.WithContext(p.ctx), t.Id())(m)
	if err != nil {
		return 0, err
	}

	return a.Id(), nil
}

func (p *ProcessorImpl) DepositAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error) {
	assetId, err := p.Deposit(worldId, accountId, body)
	if err != nil {
		return 0, err
	}

	_ = p.emitDepositedEvent(transactionId, worldId, accountId, assetId, body)

	return assetId, nil
}

func (p *ProcessorImpl) Withdraw(body message.WithdrawBody) error {
	assetId := uint32(body.AssetId)
	quantity := uint32(body.Quantity)

	a, err := asset.GetById(p.db.WithContext(p.ctx))(assetId)
	if err != nil {
		return err
	}

	if a.IsStackable() && quantity > 0 && quantity < a.Quantity() {
		return asset.UpdateQuantity(p.l, p.db.WithContext(p.ctx))(assetId, a.Quantity()-quantity)
	}

	return asset.Delete(p.l, p.db.WithContext(p.ctx))(assetId)
}

func (p *ProcessorImpl) WithdrawAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.WithdrawBody) error {
	assetId := uint32(body.AssetId)

	a, err := asset.GetById(p.db.WithContext(p.ctx))(assetId)
	if err != nil {
		return err
	}

	err = p.Withdraw(body)
	if err != nil {
		return err
	}

	_ = p.emitWithdrawnEvent(transactionId, worldId, accountId, a, body.Quantity)

	return nil
}

func (p *ProcessorImpl) UpdateMesos(worldId world.Id, accountId uint32, body message.UpdateMesosBody) error {
	s, err := GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)
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

	return UpdateMesos(p.l, p.db.WithContext(p.ctx))(s.Id(), newMesos)
}

func (p *ProcessorImpl) UpdateMesosAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.UpdateMesosBody) error {
	s, err := GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)
	if err != nil {
		return err
	}

	if body.Operation == "SUBTRACT" && s.Mesos() < body.Mesos {
		p.l.Warnf("Insufficient mesos in storage for account [%d]. Available: [%d], Requested: [%d]", accountId, s.Mesos(), body.Mesos)
		_ = p.emitErrorEvent(transactionId, worldId, accountId, message.ErrorCodeNotEnoughMesos, "Insufficient mesos in storage")
		return errors.New("insufficient mesos in storage")
	}

	oldMesos := s.Mesos()

	err = p.UpdateMesos(worldId, accountId, body)
	if err != nil {
		return err
	}

	s, _ = GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)

	_ = p.emitMesosUpdatedEvent(transactionId, worldId, accountId, oldMesos, s.Mesos())

	return nil
}

func (p *ProcessorImpl) DepositRollback(body message.DepositRollbackBody) error {
	return asset.Delete(p.l, p.db.WithContext(p.ctx))(uint32(body.AssetId))
}

// Accept accepts an item into storage as part of a transfer saga
func (p *ProcessorImpl) Accept(worldId world.Id, accountId uint32, body compartment.AcceptCommandBody) (uint32, int16, error) {
	t := tenant.MustFromContext(p.ctx)

	s, err := p.GetOrCreateStorage(worldId, accountId)
	if err != nil {
		return 0, 0, err
	}

	// Build asset model from command body
	m := asset.NewBuilder(s.Id(), body.TemplateId).
		SetExpiration(body.Expiration).
		SetQuantity(body.Quantity).
		SetOwnerId(body.OwnerId).
		SetFlag(body.Flag).
		SetRechargeable(body.Rechargeable).
		SetStrength(body.Strength).
		SetDexterity(body.Dexterity).
		SetIntelligence(body.Intelligence).
		SetLuck(body.Luck).
		SetHp(body.Hp).
		SetMp(body.Mp).
		SetWeaponAttack(body.WeaponAttack).
		SetMagicAttack(body.MagicAttack).
		SetWeaponDefense(body.WeaponDefense).
		SetMagicDefense(body.MagicDefense).
		SetAccuracy(body.Accuracy).
		SetAvoidability(body.Avoidability).
		SetHands(body.Hands).
		SetSpeed(body.Speed).
		SetJump(body.Jump).
		SetSlots(body.Slots).
		SetLevelType(body.LevelType).
		SetLevel(body.Level).
		SetExperience(body.Experience).
		SetHammersApplied(body.HammersApplied).
		SetCashId(body.CashId).
		SetCommodityId(body.CommodityId).
		SetPurchaseBy(body.PurchaseBy).
		SetPetId(body.PetId).
		Build()

	invType := inventoryTypeFromTemplateId(body.TemplateId)

	// Determine slot
	compartmentAssets, err := asset.GetByStorageIdAndInventoryType(p.db.WithContext(p.ctx))(s.Id(), invType)
	if err != nil {
		return 0, 0, err
	}
	slot := int16(len(compartmentAssets))

	// For stackable items, try to merge with existing stacks
	if m.IsStackable() {
		actualQuantity := m.Quantity()
		if actualQuantity == 0 {
			actualQuantity = 1
		}

		existingAssets, err := asset.GetByStorageIdAndTemplateId(p.db.WithContext(p.ctx))(s.Id(), body.TemplateId)
		if err == nil && len(existingAssets) > 0 {
			slotMax, err := p.getSlotMax(body.TemplateId, m)
			if err != nil || slotMax == 0 {
				slotMax = 100
			}

			for _, existing := range existingAssets {
				if existing.Rechargeable() > 0 {
					continue
				}
				if existing.OwnerId() != m.OwnerId() || existing.Flag() != m.Flag() {
					continue
				}
				if existing.Quantity()+actualQuantity <= slotMax {
					newQuantity := existing.Quantity() + actualQuantity
					err = asset.UpdateQuantity(p.l, p.db.WithContext(p.ctx))(existing.Id(), newQuantity)
					if err != nil {
						continue
					}
					return existing.Id(), existing.Slot(), nil
				}
			}
		}
	}

	// No merge — create new asset
	m = asset.Clone(m).SetSlot(slot).Build()
	a, err := asset.Create(p.l, p.db.WithContext(p.ctx), t.Id())(m)
	if err != nil {
		return 0, 0, err
	}

	return a.Id(), slot, nil
}

func (p *ProcessorImpl) AcceptAndEmit(worldId world.Id, accountId uint32, characterId uint32, body compartment.AcceptCommandBody) error {
	assetId, slot, err := p.Accept(worldId, accountId, body)
	if err != nil {
		_ = p.emitCompartmentErrorEvent(worldId, accountId, characterId, body.TransactionId, "ACCEPT_FAILED", err.Error())
		return err
	}

	invType := inventoryTypeFromTemplateId(body.TemplateId)

	return p.emitCompartmentAcceptedEvent(worldId, accountId, characterId, body.TransactionId, assetId, slot, invType)
}

func (p *ProcessorImpl) Release(body compartment.ReleaseCommandBody) error {
	assetId := uint32(body.AssetId)
	quantity := uint32(body.Quantity)

	a, err := asset.GetById(p.db.WithContext(p.ctx))(assetId)
	if err != nil {
		return err
	}

	if quantity == 0 || !a.IsStackable() {
		return asset.Delete(p.l, p.db.WithContext(p.ctx))(assetId)
	}

	if quantity >= a.Quantity() {
		return asset.Delete(p.l, p.db.WithContext(p.ctx))(assetId)
	}

	newQuantity := a.Quantity() - quantity
	return asset.UpdateQuantity(p.l, p.db.WithContext(p.ctx))(assetId, newQuantity)
}

func (p *ProcessorImpl) ReleaseAndEmit(worldId world.Id, accountId uint32, characterId uint32, body compartment.ReleaseCommandBody) error {
	assetId := uint32(body.AssetId)

	a, err := asset.GetById(p.db.WithContext(p.ctx))(assetId)
	if err != nil {
		_ = p.emitCompartmentErrorEvent(worldId, accountId, characterId, body.TransactionId, "RELEASE_FAILED", err.Error())
		return err
	}

	invType := inventoryTypeFromTemplateId(a.TemplateId())

	err = p.Release(body)
	if err != nil {
		_ = p.emitCompartmentErrorEvent(worldId, accountId, characterId, body.TransactionId, "RELEASE_FAILED", err.Error())
		return err
	}

	return p.emitCompartmentReleasedEvent(worldId, accountId, characterId, body.TransactionId, body.AssetId, invType)
}

func inventoryTypeFromTemplateId(templateId uint32) byte {
	t, _ := inventory.TypeFromItemId(item.Id(templateId))
	return byte(t)
}

func (p *ProcessorImpl) emitCompartmentAcceptedEvent(worldId world.Id, accountId uint32, characterId uint32, transactionId uuid.UUID, assetId uint32, slot int16, inventoryType byte) error {
	event := &compartment.StatusEvent[compartment.StatusEventAcceptedBody]{
		WorldId:     worldId,
		AccountId:   accountId,
		CharacterId: characterId,
		Type:        compartment.StatusEventTypeAccepted,
		Body: compartment.StatusEventAcceptedBody{
			TransactionId: transactionId,
			AssetId:       assetConstants.Id(assetId),
			Slot:          slot,
			InventoryType: inventoryType,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvEventTopicStatus)(createCompartmentMessageProvider(accountId, event))
}

func (p *ProcessorImpl) emitCompartmentReleasedEvent(worldId world.Id, accountId uint32, characterId uint32, transactionId uuid.UUID, assetId assetConstants.Id, inventoryType byte) error {
	event := &compartment.StatusEvent[compartment.StatusEventReleasedBody]{
		WorldId:     worldId,
		AccountId:   accountId,
		CharacterId: characterId,
		Type:        compartment.StatusEventTypeReleased,
		Body: compartment.StatusEventReleasedBody{
			TransactionId: transactionId,
			AssetId:       assetId,
			InventoryType: inventoryType,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvEventTopicStatus)(createCompartmentMessageProvider(accountId, event))
}

func (p *ProcessorImpl) emitCompartmentErrorEvent(worldId world.Id, accountId uint32, characterId uint32, transactionId uuid.UUID, errorCode string, errorMessage string) error {
	event := &compartment.StatusEvent[compartment.StatusEventErrorBody]{
		WorldId:     worldId,
		AccountId:   accountId,
		CharacterId: characterId,
		Type:        compartment.StatusEventTypeError,
		Body: compartment.StatusEventErrorBody{
			TransactionId: transactionId,
			ErrorCode:     errorCode,
			Message:       errorMessage,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(compartment.EnvEventTopicStatus)(createCompartmentMessageProvider(accountId, event))
}

func createCompartmentMessageProvider[E any](accountId uint32, event *compartment.StatusEvent[E]) func() ([]kafka.Message, error) {
	key := producer.CreateKey(int(accountId))
	return producer.SingleMessageProvider(key, event)
}

func (p *ProcessorImpl) emitDepositedEvent(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId uint32, body message.DepositBody) error {
	event := &message.StatusEvent[message.DepositedEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeDeposited,
		Body: message.DepositedEventBody{
			AssetId:    assetConstants.Id(assetId),
			Slot:       body.Slot,
			TemplateId: body.TemplateId,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func (p *ProcessorImpl) emitWithdrawnEvent(transactionId uuid.UUID, worldId world.Id, accountId uint32, a asset.Model, quantity assetConstants.Quantity) error {
	event := &message.StatusEvent[message.WithdrawnEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeWithdrawn,
		Body: message.WithdrawnEventBody{
			AssetId:    assetConstants.Id(a.Id()),
			Slot:       a.Slot(),
			TemplateId: a.TemplateId(),
			Quantity:   quantity,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func (p *ProcessorImpl) emitMesosUpdatedEvent(transactionId uuid.UUID, worldId world.Id, accountId uint32, oldMesos uint32, newMesos uint32) error {
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
	key := producer.CreateKey(int(accountId))
	return producer.SingleMessageProvider(key, event)
}

// mergeKey is used to group stackable assets for merging
type mergeKey struct {
	templateId uint32
	ownerId    uint32
	flag       uint16
}

func (p *ProcessorImpl) MergeAndSort(worldId world.Id, accountId uint32) error {
	s, err := GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)
	if err != nil {
		return err
	}

	assets, err := asset.GetByStorageId(p.db.WithContext(p.ctx))(s.Id())
	if err != nil {
		return err
	}

	var nonStackables []asset.Model
	stackableGroups := make(map[mergeKey][]asset.Model)

	for _, a := range assets {
		if !a.IsStackable() {
			nonStackables = append(nonStackables, a)
			continue
		}

		// Check if consumable is rechargeable (cannot merge)
		if a.IsConsumable() && a.Rechargeable() > 0 {
			nonStackables = append(nonStackables, a)
			continue
		}

		key := mergeKey{
			templateId: a.TemplateId(),
			ownerId:    a.OwnerId(),
			flag:       a.Flag(),
		}
		stackableGroups[key] = append(stackableGroups[key], a)
	}

	// Prefetch slot maxima before opening the transaction — these are
	// atlas-data lookups (network I/O) and must not run inside the tx.
	slotMaxByTemplate := make(map[uint32]uint32, len(stackableGroups))
	for key := range stackableGroups {
		if _, seen := slotMaxByTemplate[key.templateId]; seen {
			continue
		}
		slotMax, err := p.getSlotMaxByTemplateId(key.templateId)
		if err != nil || slotMax == 0 {
			slotMax = 100
		}
		slotMaxByTemplate[key.templateId] = slotMax
	}

	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		if len(stackableGroups) == 0 {
			return p.sortAssets(tx, assets)
		}

		var mergedAssets []asset.Model

		for key, group := range stackableGroups {
			slotMax := slotMaxByTemplate[key.templateId]

			var totalQuantity uint32
			for _, a := range group {
				totalQuantity += a.Quantity()
			}

			numStacks := (totalQuantity + slotMax - 1) / slotMax
			if numStacks == 0 {
				numStacks = 1
			}

			assetsToKeep := min(uint32(len(group)), numStacks)

			sort.Slice(group, func(i, j int) bool {
				return group[i].Slot() < group[j].Slot()
			})

			remainingQuantity := totalQuantity
			for i := uint32(0); i < assetsToKeep; i++ {
				a := group[i]
				newQuantity := min(remainingQuantity, slotMax)
				remainingQuantity -= newQuantity

				if err := asset.UpdateQuantity(p.l, tx)(a.Id(), newQuantity); err != nil {
					return err
				}

				mergedAssets = append(mergedAssets, a)
			}

			for i := int(assetsToKeep); i < len(group); i++ {
				if err := asset.Delete(p.l, tx)(group[i].Id()); err != nil {
					return err
				}
			}
		}

		allAssets := append(nonStackables, mergedAssets...)

		return p.sortAssets(tx, allAssets)
	})
}

func (p *ProcessorImpl) sortAssets(db *gorm.DB, assets []asset.Model) error {
	byInventoryType := make(map[byte][]asset.Model)
	for _, a := range assets {
		invType := inventoryTypeFromTemplateId(a.TemplateId())
		byInventoryType[invType] = append(byInventoryType[invType], a)
	}

	for it := range byInventoryType {
		group := byInventoryType[it]
		sort.Slice(group, func(i, j int) bool {
			return group[i].TemplateId() < group[j].TemplateId()
		})
		byInventoryType[it] = group
	}

	for _, group := range byInventoryType {
		for i, a := range group {
			newSlot := int16(i)
			if a.Slot() != newSlot {
				if err := asset.UpdateSlot(p.l, db)(a.Id(), newSlot); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (p *ProcessorImpl) getSlotMax(templateId uint32, m asset.Model) (uint32, error) {
	if m.IsConsumable() {
		cp := consumable.NewProcessor(p.l, p.ctx)
		c, err := cp.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return c.SlotMax(), nil
	}
	if m.IsSetup() {
		sp := setup.NewProcessor(p.l, p.ctx)
		s, err := sp.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return s.SlotMax(), nil
	}
	if m.IsEtc() {
		ep := etc.NewProcessor(p.l, p.ctx)
		e, err := ep.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return e.SlotMax(), nil
	}
	return 0, nil
}

func (p *ProcessorImpl) getSlotMaxByTemplateId(templateId uint32) (uint32, error) {
	invType := inventoryTypeFromTemplateId(templateId)
	switch inventory.Type(invType) {
	case inventory.TypeValueUse:
		cp := consumable.NewProcessor(p.l, p.ctx)
		c, err := cp.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return c.SlotMax(), nil
	case inventory.TypeValueSetup:
		sp := setup.NewProcessor(p.l, p.ctx)
		s, err := sp.ByIdProvider(templateId)()
		if err != nil {
			return 0, err
		}
		return s.SlotMax(), nil
	case inventory.TypeValueETC:
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

func (p *ProcessorImpl) ArrangeAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32) error {
	err := p.MergeAndSort(worldId, accountId)
	if err != nil {
		_ = p.emitErrorEvent(transactionId, worldId, accountId, message.ErrorCodeGeneric, err.Error())
		return err
	}

	return p.emitArrangedEvent(transactionId, worldId, accountId)
}

func (p *ProcessorImpl) emitArrangedEvent(transactionId uuid.UUID, worldId world.Id, accountId uint32) error {
	event := &message.StatusEvent[message.ArrangedEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeArranged,
		Body:          message.ArrangedEventBody{},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func (p *ProcessorImpl) emitErrorEvent(transactionId uuid.UUID, worldId world.Id, accountId uint32, errorCode string, errorMessage string) error {
	event := &message.StatusEvent[message.ErrorEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeError,
		Body: message.ErrorEventBody{
			ErrorCode: errorCode,
			Message:   errorMessage,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}

func (p *ProcessorImpl) EmitProjectionCreatedEvent(characterId uint32, accountId uint32, ch channel.Model, npcId uint32) error {
	event := &message.StatusEvent[message.ProjectionCreatedEventBody]{
		WorldId:   ch.WorldId(),
		AccountId: accountId,
		Type:      message.StatusEventTypeProjectionCreated,
		Body: message.ProjectionCreatedEventBody{
			CharacterId: characterId,
			AccountId:   accountId,
			WorldId:     ch.WorldId(),
			ChannelId:   ch.Id(),
			NpcId:       npcId,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func (p *ProcessorImpl) ExpireAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId uint32, isCash bool, replaceItemId uint32, replaceMessage string) error {
	t := tenant.MustFromContext(p.ctx)

	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		a, err := asset.GetById(tx)(assetId)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to find asset [%d] for expiration.", assetId)
			return err
		}

		if err := asset.Delete(p.l, tx)(assetId); err != nil {
			p.l.WithError(err).Errorf("Failed to delete expired asset [%d].", assetId)
			return err
		}

		if replaceItemId > 0 {
			p.l.Debugf("Creating replacement item [%d] for expired storage item [%d].", replaceItemId, a.TemplateId())

			s, err := p.WithTransaction(tx).GetOrCreateStorage(worldId, accountId)
			if err != nil {
				p.l.WithError(err).Errorf("Failed to get storage for replacement item creation.")
				return err
			}

			assets, err := asset.GetByStorageId(tx)(s.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Failed to get assets for slot calculation.")
				return err
			}
			nextSlot := int16(len(assets))

			replacement := asset.NewBuilder(s.Id(), replaceItemId).
				SetSlot(nextSlot).
				Build()

			if _, err := asset.Create(p.l, tx, t.Id())(replacement); err != nil {
				p.l.WithError(err).Errorf("Failed to create replacement item [%d] for account [%d].", replaceItemId, accountId)
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Publish only after the transaction commits: no event for a rolled-back expiry.
	_ = p.emitExpiredEvent(transactionId, worldId, accountId, isCash, replaceItemId, replaceMessage)

	p.l.Debugf("Expired asset [%d] from storage for account [%d].", assetId, accountId)
	return nil
}

func (p *ProcessorImpl) emitExpiredEvent(transactionId uuid.UUID, worldId world.Id, accountId uint32, isCash bool, replaceItemId uint32, replaceMessage string) error {
	event := &message.StatusEvent[message.ExpiredStatusEventBody]{
		TransactionId: transactionId,
		WorldId:       worldId,
		AccountId:     accountId,
		Type:          message.StatusEventTypeExpired,
		Body: message.ExpiredStatusEventBody{
			IsCash:         isCash,
			ReplaceItemId:  replaceItemId,
			ReplaceMessage: replaceMessage,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}

func (p *ProcessorImpl) DeleteByAccountId(accountId uint32) error {
	storages, err := GetByAccountId(p.l, p.db.WithContext(p.ctx))(accountId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to retrieve storages for account [%d].", accountId)
		return err
	}

	p.l.Infof("Deleting [%d] storage(s) for account [%d].", len(storages), accountId)

	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		for _, s := range storages {
			if err := asset.DeleteByStorageId(p.l, tx)(s.Id()); err != nil {
				p.l.WithError(err).Errorf("Failed to delete assets for storage [%s].", s.Id())
				return err
			}
			if err := Delete(p.l, tx)(s.Id()); err != nil {
				p.l.WithError(err).Errorf("Failed to delete storage [%s] for account [%d].", s.Id(), accountId)
				return err
			}
		}
		return nil
	})
}

func (p *ProcessorImpl) EmitProjectionDestroyedEvent(characterId uint32, accountId uint32, worldId world.Id) error {
	event := &message.StatusEvent[message.ProjectionDestroyedEventBody]{
		WorldId:   worldId,
		AccountId: accountId,
		Type:      message.StatusEventTypeProjectionDestroyed,
		Body: message.ProjectionDestroyedEventBody{
			CharacterId: characterId,
		},
	}

	return producer.ProviderImpl(p.l)(p.ctx)(message.EnvEventTopic)(createMessageProvider(accountId, event))
}
