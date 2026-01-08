package handler

import (
	"atlas-channel/compartment"
	npcTemplate "atlas-channel/data/npc/template"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/storage"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

type StorageOperationMode string

const (
	StorageOperationHandle       = "StorageOperationHandle"
	StorageOperationModeRetrieve = "RETRIEVE_ASSET" // 4
	StorageOperationModeStore    = "STORE_ASSET"    // 5
	StorageOperationModeArrange  = "ARRANGE_ASSET"  // 6
	StorageOperationModeMeso     = "MESO"           // 7
	StorageOperationModeClose    = "CLOSE"
)

func StorageOperationHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		mode := r.ReadByte()
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeRetrieve) {
			handleRetrieveAsset(l, ctx, s, r)
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeStore) {
			handleStoreAsset(l, ctx, s, r)
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeArrange) {
			handleArrangeAsset(l, ctx, s)
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeMeso) {
			handleMeso(l, ctx, s, r)
			return
		}
		if isStorageOperation(l)(readerOptions, mode, StorageOperationModeClose) {
			handleClose(l, ctx, s)
			return
		}
	}
}

// getStorageNpcFees retrieves the storage fees for the current NPC
func getStorageNpcFees(l logrus.FieldLogger, ctx context.Context, npcId uint32) (depositFee int32, withdrawFee int32) {
	if npcId == 0 {
		return 0, 0
	}

	npc, err := npcTemplate.NewProcessor(l, ctx).GetById(npcId)
	if err != nil {
		l.WithError(err).Warnf("Unable to get NPC [%d] for storage fees, using free storage.", npcId)
		return 0, 0
	}

	return npc.GetDepositFee(), npc.GetWithdrawFee()
}

// handleRetrieveAsset withdraws an item from storage to character inventory
func handleRetrieveAsset(l logrus.FieldLogger, ctx context.Context, s session.Model, r *request.Reader) {
	it := inventory.Type(r.ReadByte())
	slot := r.ReadByte()
	l.Debugf("Character [%d] is attempting to retrieve an item from storage inventory type [%d] slot [%d].", s.CharacterId(), it, slot)

	// Get storage NPC fees
	_, withdrawFee := getStorageNpcFees(l, ctx, s.StorageNpcId())
	if withdrawFee > 0 {
		l.Debugf("Storage withdrawal fee for NPC [%d]: %d mesos", s.StorageNpcId(), withdrawFee)
		// TODO: Deduct fee from character's mesos before transfer
		// This requires integration with the character meso system
	}

	// Get the compartment for the target inventory type
	cp := compartment.NewProcessor(l, ctx)
	targetCompartment, err := cp.GetByType(s.CharacterId(), it)
	if err != nil {
		l.WithError(err).Errorf("Unable to get compartment for character [%d] inventory type [%d].", s.CharacterId(), it)
		return
	}

	// Get the asset from storage by slot
	storageData, err := storage.NewProcessor(l, ctx).GetStorageData(s.AccountId(), byte(s.WorldId()))
	if err != nil {
		l.WithError(err).Errorf("Unable to get storage data for account [%d].", s.AccountId())
		return
	}

	// Find the asset at the given slot
	var assetId uint32
	var referenceId uint32
	found := false
	for _, a := range storageData.Assets {
		if a.Slot() == int16(slot) {
			assetId = a.Id()
			referenceId = a.ReferenceId()
			found = true
			break
		}
	}

	if !found {
		l.Warnf("No asset found at storage slot [%d] for account [%d].", slot, s.AccountId())
		return
	}

	// Transfer asset from storage to character inventory
	err = cp.TransferFromStorage(byte(s.WorldId()), s.AccountId(), s.CharacterId(), assetId, targetCompartment.Id(), byte(it), referenceId)
	if err != nil {
		l.WithError(err).Errorf("Unable to transfer asset [%d] from storage to character [%d].", assetId, s.CharacterId())
		return
	}
}

// handleStoreAsset deposits an item from character inventory to storage
func handleStoreAsset(l logrus.FieldLogger, ctx context.Context, s session.Model, r *request.Reader) {
	slot := r.ReadInt16()
	itemId := r.ReadUint32()
	quantity := r.ReadUint16()
	l.Debugf("Character [%d] is attempting to store [%d] of item [%d] from inventory slot [%d].", s.CharacterId(), quantity, itemId, slot)

	// Get storage NPC fees
	depositFee, _ := getStorageNpcFees(l, ctx, s.StorageNpcId())
	if depositFee > 0 {
		l.Debugf("Storage deposit fee for NPC [%d]: %d mesos", s.StorageNpcId(), depositFee)
		// TODO: Deduct fee from character's mesos before transfer
		// This requires integration with the character meso system
	}

	// Determine inventory type from itemId
	it, ok := inventory.TypeFromItemId(item.Id(itemId))
	if !ok {
		l.Warnf("Unable to determine inventory type from item [%d].", itemId)
		return
	}

	// Get the source compartment
	cp := compartment.NewProcessor(l, ctx)
	sourceCompartment, err := cp.GetByType(s.CharacterId(), it)
	if err != nil {
		l.WithError(err).Errorf("Unable to get compartment for character [%d] inventory type [%d].", s.CharacterId(), it)
		return
	}

	// Get the asset from the source compartment at the given slot
	// For now, we pass the slot as the assetId placeholder since we need to look it up
	// The compartment-transfer service will look up the actual asset

	// Get reference type from inventory type
	var refType string
	switch it {
	case inventory.TypeValueEquip:
		refType = "EQUIPABLE"
	case inventory.TypeValueUse:
		refType = "CONSUMABLE"
	case inventory.TypeValueSetup:
		refType = "SETUP"
	case inventory.TypeValueETC:
		refType = "ETC"
	case inventory.TypeValueCash:
		refType = "CASH"
	default:
		refType = "ETC"
	}

	// Transfer asset from character inventory to storage
	// Note: assetId and referenceId need to be looked up from the character's inventory
	// For now we pass 0 - the actual implementation would need to look up the asset
	err = cp.TransferToStorage(byte(s.WorldId()), s.AccountId(), s.CharacterId(), 0, sourceCompartment.Id(), byte(it), 0, itemId, refType, slot)
	if err != nil {
		l.WithError(err).Errorf("Unable to transfer asset from inventory slot [%d] to storage for character [%d].", slot, s.CharacterId())
		return
	}
}

// handleArrangeAsset arranges (merges and sorts) items in storage
func handleArrangeAsset(l logrus.FieldLogger, ctx context.Context, s session.Model) {
	l.Debugf("Character [%d] would like to arrange their storage.", s.CharacterId())

	err := storage.NewProcessor(l, ctx).Arrange(byte(s.WorldId()), s.AccountId())
	if err != nil {
		l.WithError(err).Errorf("Unable to arrange storage for account [%d].", s.AccountId())
	}
}

// handleMeso handles meso deposit/withdrawal operations
func handleMeso(l logrus.FieldLogger, ctx context.Context, s session.Model, r *request.Reader) {
	amount := r.ReadInt32()
	sp := storage.NewProcessor(l, ctx)

	if amount < 0 {
		// Negative amount = deposit mesos to storage
		mesos := uint32(-amount)
		l.Debugf("Character [%d] is attempting to deposit [%d] mesos to storage.", s.CharacterId(), mesos)
		err := sp.DepositMesos(byte(s.WorldId()), s.AccountId(), mesos)
		if err != nil {
			l.WithError(err).Errorf("Unable to deposit [%d] mesos to storage for account [%d].", mesos, s.AccountId())
		}
	} else if amount > 0 {
		// Positive amount = withdraw mesos from storage
		mesos := uint32(amount)
		l.Debugf("Character [%d] is attempting to withdraw [%d] mesos from storage.", s.CharacterId(), mesos)
		err := sp.WithdrawMesos(byte(s.WorldId()), s.AccountId(), mesos)
		if err != nil {
			l.WithError(err).Errorf("Unable to withdraw [%d] mesos from storage for account [%d].", mesos, s.AccountId())
		}
	}
}

// handleClose handles closing the storage UI
func handleClose(l logrus.FieldLogger, ctx context.Context, s session.Model) {
	l.Debugf("Character [%d] exited storage.", s.CharacterId())
	// Clear the storage NPC ID from the session
	session.NewProcessor(l, ctx).ClearStorageNpcId(s.SessionId())
}

func isStorageOperation(l logrus.FieldLogger) func(options map[string]interface{}, op byte, key StorageOperationMode) bool {
	return func(options map[string]interface{}, op byte, key StorageOperationMode) bool {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["operations"]; !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}

		res, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use.", key)
			return false
		}
		return byte(res) == op
	}
}
