package handler

import (
	npcTemplate "atlas-channel/data/npc/template"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"atlas-channel/storage"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type StorageOperationMode string

const (
	StorageOperationHandle       = "StorageOperationHandle"
	StorageOperationModeRetrieve = "RETRIEVE_ASSET" // 4
	StorageOperationModeStore    = "STORE_ASSET"    // 5
	StorageOperationModeArrange  = "ARRANGE_ASSET"  // 6
	StorageOperationModeMeso     = "MESO"           // 7
	StorageOperationModeClose    = "CLOSE"          // 8
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

// handleRetrieveAsset withdraws an item from storage to character inventory via saga
func handleRetrieveAsset(l logrus.FieldLogger, ctx context.Context, s session.Model, r *request.Reader) {
	it := inventory.Type(r.ReadByte())
	slot := r.ReadByte()
	l.Debugf("Character [%d] is attempting to retrieve an item from storage inventory type [%d] slot [%d].", s.CharacterId(), it, slot)

	// Get storage NPC fees
	_, withdrawFee := getStorageNpcFees(l, ctx, s.StorageNpcId())

	// Create saga transaction
	sagaP := saga.NewProcessor(l, ctx)
	transactionId := uuid.New()
	now := time.Now()

	// Build saga steps
	steps := make([]saga.Step[any], 0, 2)

	// Step 1: Charge withdrawal fee (if applicable)
	if withdrawFee > 0 {
		l.Debugf("Storage withdrawal fee for NPC [%d]: %d mesos", s.StorageNpcId(), withdrawFee)
		steps = append(steps, saga.Step[any]{
			StepId: "charge_withdrawal_fee",
			Status: saga.Pending,
			Action: saga.AwardMesos,
			Payload: saga.AwardMesosPayload{
				WorldId:     s.WorldId(),
				CharacterId: s.CharacterId(),
				ChannelId:   s.ChannelId(),
				ActorId:     s.StorageNpcId(),
				ActorType:   "NPC",
				Amount:      -withdrawFee, // Negative to deduct
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Step 2: High-level withdrawal step (will be expanded by saga-orchestrator)
	steps = append(steps, saga.Step[any]{
		StepId: "withdraw_from_storage",
		Status: saga.Pending,
		Action: saga.WithdrawFromStorage,
		Payload: saga.WithdrawFromStoragePayload{
			TransactionId: transactionId,
			CharacterId:   s.CharacterId(),
			WorldId:       s.WorldId(),
			AccountId:     s.AccountId(),
			SourceSlot:    int16(slot),
			InventoryType: byte(it),
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Create saga transaction
	sagaTx := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.StorageOperation,
		InitiatedBy:   "STORAGE",
		Steps:         steps,
	}

	err := sagaP.Create(sagaTx)
	if err != nil {
		l.WithError(err).Errorf("Unable to create saga for withdrawing from storage slot [%d] for character [%d].", slot, s.CharacterId())
	} else {
		l.Debugf("Created withdrawal saga [%s] for character [%d] withdrawing from storage slot [%d].", transactionId.String(), s.CharacterId(), slot)
	}
}

// handleStoreAsset deposits an item from character inventory to storage via saga
func handleStoreAsset(l logrus.FieldLogger, ctx context.Context, s session.Model, r *request.Reader) {
	slot := r.ReadInt16()
	itemId := r.ReadUint32()
	quantity := r.ReadUint16()
	l.Debugf("Character [%d] is attempting to store [%d] of item [%d] from inventory slot [%d].", s.CharacterId(), quantity, itemId, slot)

	// Get storage NPC fees
	depositFee, _ := getStorageNpcFees(l, ctx, s.StorageNpcId())

	// Determine inventory type from itemId
	it, ok := inventory.TypeFromItemId(item.Id(itemId))
	if !ok {
		l.Warnf("Unable to determine inventory type from item [%d].", itemId)
		return
	}

	// Create saga transaction
	sagaP := saga.NewProcessor(l, ctx)
	transactionId := uuid.New()
	now := time.Now()

	// Build saga steps
	steps := make([]saga.Step[any], 0, 2)

	// Step 1: Charge deposit fee (if applicable)
	if depositFee > 0 {
		l.Debugf("Storage deposit fee for NPC [%d]: %d mesos", s.StorageNpcId(), depositFee)
		steps = append(steps, saga.Step[any]{
			StepId: "charge_deposit_fee",
			Status: saga.Pending,
			Action: saga.AwardMesos,
			Payload: saga.AwardMesosPayload{
				WorldId:     s.WorldId(),
				CharacterId: s.CharacterId(),
				ChannelId:   s.ChannelId(),
				ActorId:     s.StorageNpcId(),
				ActorType:   "NPC",
				Amount:      -depositFee, // Negative to deduct
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Step 2: High-level transfer step (will be expanded by saga-orchestrator)
	steps = append(steps, saga.Step[any]{
		StepId: "transfer_to_storage",
		Status: saga.Pending,
		Action: saga.TransferToStorage,
		Payload: saga.TransferToStoragePayload{
			TransactionId:       transactionId,
			CharacterId:         s.CharacterId(),
			WorldId:             s.WorldId(),
			AccountId:           s.AccountId(),
			SourceSlot:          slot,
			SourceInventoryType: byte(it),
			Quantity:            uint32(quantity),
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	// Create saga transaction
	sagaTx := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.StorageOperation,
		InitiatedBy:   "STORAGE",
		Steps:         steps,
	}

	err := sagaP.Create(sagaTx)
	if err != nil {
		l.WithError(err).Errorf("Unable to create saga for depositing item [%d] to storage for character [%d].", itemId, s.CharacterId())
	} else {
		l.Debugf("Created deposit saga [%s] for character [%d] depositing item [%d].", transactionId.String(), s.CharacterId(), itemId)
	}
}

// handleArrangeAsset arranges (merges and sorts) items in storage
func handleArrangeAsset(l logrus.FieldLogger, ctx context.Context, s session.Model) {
	l.Debugf("Character [%d] would like to arrange their storage.", s.CharacterId())

	err := storage.NewProcessor(l, ctx).Arrange(s.WorldId(), s.AccountId())
	if err != nil {
		l.WithError(err).Errorf("Unable to arrange storage for account [%d].", s.AccountId())
	}
}

// handleMeso handles meso deposit/withdrawal operations via saga
func handleMeso(l logrus.FieldLogger, ctx context.Context, s session.Model, r *request.Reader) {
	amount := r.ReadInt32()
	sagaP := saga.NewProcessor(l, ctx)
	transactionId := uuid.New()
	now := time.Now()

	if amount < 0 {
		// Negative amount = deposit mesos to storage
		// Saga: 1) Deduct from character, 2) Add to storage
		mesos := uint32(-amount)
		l.Debugf("Character [%d] is attempting to deposit [%d] mesos to storage via saga [%s].", s.CharacterId(), mesos, transactionId.String())

		// Step 1: Deduct mesos from character (negative amount)
		step1 := saga.Step[any]{
			StepId: "deduct_character_mesos",
			Status: saga.Pending,
			Action: saga.AwardMesos,
			Payload: saga.AwardMesosPayload{
				WorldId:     s.WorldId(),
				CharacterId: s.CharacterId(),
				ChannelId:   s.ChannelId(),
				ActorId:     s.CharacterId(),
				ActorType:   "STORAGE",
				Amount:      amount, // Negative to deduct
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Step 2: Add mesos to storage
		step2 := saga.Step[any]{
			StepId: "add_storage_mesos",
			Status: saga.Pending,
			Action: saga.UpdateStorageMesos,
			Payload: saga.UpdateStorageMesosPayload{
				CharacterId: s.CharacterId(),
				AccountId:   s.AccountId(),
				WorldId:     s.WorldId(),
				Operation:   "ADD",
				Mesos:       mesos,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		sagaTx := saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.StorageOperation,
			InitiatedBy:   "STORAGE",
			Steps:         []saga.Step[any]{step1, step2},
		}

		err := sagaP.Create(sagaTx)
		if err != nil {
			l.WithError(err).Errorf("Unable to create saga for depositing [%d] mesos to storage for character [%d].", mesos, s.CharacterId())
		}
	} else if amount > 0 {
		// Positive amount = withdraw mesos from storage
		// Saga: 1) Deduct from storage, 2) Add to character
		mesos := uint32(amount)
		l.Debugf("Character [%d] is attempting to withdraw [%d] mesos from storage via saga [%s].", s.CharacterId(), mesos, transactionId.String())

		// Step 1: Deduct mesos from storage
		step1 := saga.Step[any]{
			StepId: "subtract_storage_mesos",
			Status: saga.Pending,
			Action: saga.UpdateStorageMesos,
			Payload: saga.UpdateStorageMesosPayload{
				CharacterId: s.CharacterId(),
				AccountId:   s.AccountId(),
				WorldId:     s.WorldId(),
				Operation:   "SUBTRACT",
				Mesos:       mesos,
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Step 2: Add mesos to character
		step2 := saga.Step[any]{
			StepId: "add_character_mesos",
			Status: saga.Pending,
			Action: saga.AwardMesos,
			Payload: saga.AwardMesosPayload{
				WorldId:     s.WorldId(),
				CharacterId: s.CharacterId(),
				ChannelId:   s.ChannelId(),
				ActorId:     s.CharacterId(),
				ActorType:   "STORAGE",
				Amount:      amount, // Positive to add
			},
			CreatedAt: now,
			UpdatedAt: now,
		}

		sagaTx := saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.StorageOperation,
			InitiatedBy:   "STORAGE",
			Steps:         []saga.Step[any]{step1, step2},
		}

		err := sagaP.Create(sagaTx)
		if err != nil {
			l.WithError(err).Errorf("Unable to create saga for withdrawing [%d] mesos from storage for character [%d].", mesos, s.CharacterId())
		}
	}
}

// handleClose handles closing the storage UI
func handleClose(l logrus.FieldLogger, ctx context.Context, s session.Model) {
	l.Debugf("Character [%d] exited storage.", s.CharacterId())
	// Emit close storage command to clear NPC context cache in storage service
	err := storage.NewProcessor(l, ctx).CloseStorage(s.CharacterId())
	if err != nil {
		l.WithError(err).Errorf("Unable to emit close storage command for character [%d].", s.CharacterId())
	}
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
