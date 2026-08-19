package handler

import (
	"atlas-channel/asset"
	"atlas-channel/character"
	"atlas-channel/compartment"
	dueyparcel "atlas-channel/parcel"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	parcelcb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/clientbound"
	parcelsb "github.com/Chronicle20/atlas/libs/atlas-packet/parcel/serverbound"
)

const (
	// dueyReceivableDelay mirrors atlas-parcel's ReceivableDelay
	// (services/atlas-parcel/atlas.com/parcel/parcel/entity.go) — how long
	// a normal (non-return-leg) parcel sits in transit before it becomes
	// receivable. Duplicated rather than imported: atlas-parcel and
	// atlas-channel are separate Go modules, and this value is computed at
	// send time, here, before the parcel row exists (design §4.3).
	dueyReceivableDelay = 24 * time.Hour
	// dueyExpiryWindow mirrors atlas-parcel's ExpiryWindow.
	dueyExpiryWindow = 30 * 24 * time.Hour
)

// dueySendDeps collects every external lookup the DUEY_ACTION SEND
// pre-flight makes, so sendParcel can be exercised without a REST client, a
// Kafka producer or a session registry — mirroring claimSubmitDeps
// (claim_request.go).
type dueySendDeps struct {
	getSender        func(characterId uint32) (character.Model, error)
	resolveRecipient func(name string) ([]character.Model, error)
	getItem          func(characterId uint32, it inventory.Type, slot int16) (asset.Model, error)
	hasTicket        func(characterId uint32) (bool, error)
	countPending     func(recipientId uint32, worldId world.Id) (int, error)
	createSaga       func(s saga.Saga) error
}

// handleDueyActionSend wires sendParcel's dependencies to the real
// atlas-character / atlas-compartment / atlas-parcel / saga-orchestrator
// collaborators.
func handleDueyActionSend(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, sp *parcelsb.ActionSend) {
	return func(s session.Model, sp *parcelsb.ActionSend) {
		deps := dueySendDeps{
			getSender: func(characterId uint32) (character.Model, error) {
				return character.NewProcessor(l, ctx).GetById()(characterId)
			},
			resolveRecipient: func(name string) ([]character.Model, error) {
				return character.NewProcessor(l, ctx).ByNameProvider(name)()
			},
			getItem: func(characterId uint32, it inventory.Type, slot int16) (asset.Model, error) {
				return character.NewProcessor(l, ctx).GetItemInSlot(characterId, it, slot)()
			},
			hasTicket: func(characterId uint32) (bool, error) {
				cp, err := compartment.NewProcessor(l, ctx).GetByType(characterId, inventory.TypeValueETC)
				if err != nil {
					return false, err
				}
				_, found := cp.FindFirstByItemId(item.QuickDeliveryTicketId)
				return found, nil
			},
			countPending: func(recipientId uint32, worldId world.Id) (int, error) {
				return dueyparcel.NewProcessor(l, ctx).CountPending(recipientId, worldId)
			},
			createSaga: func(sg saga.Saga) error {
				return saga.NewProcessor(l, ctx).Create(sg)
			},
		}
		sendParcel(l, ctx, wp, s, sp, deps)
	}
}

// sendParcel runs the DUEY_ACTION SEND pre-flight in design §6.2's order
// (local checks, then recipient resolution, then same-account, then
// mailbox capacity, then the ticket check) and, if every check passes,
// builds the parcel_send saga. Every rejection announces a PARCEL result
// arm inline and starts no saga — mirroring note_send's posture (NFR-5:
// never a disconnect, even on a packet-edited request).
func sendParcel(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, sp *parcelsb.ActionSend, deps dueySendDeps) {
	reject := func(body func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte) {
		_ = session.Announce(l)(ctx)(wp)(parcelcb.ParcelWriter)(body)(s)
	}

	sender, err := deps.getSender(s.CharacterId())
	if err != nil {
		l.WithError(err).Errorf("Character [%d] DUEY_ACTION SEND: unable to load own character.", s.CharacterId())
		reject(parcelcb.ParcelIncorrectRequestBody())
		return
	}

	// Local-only checks (Task 16's ValidateSend) before any remote lookup.
	reason := dueyparcel.ValidateSend(dueyparcel.SendInput{
		MesoAmount:  sp.Mesos(),
		Quantity:    sp.Quantity(),
		Quick:       sp.Quick(),
		Message:     sp.Message(),
		SenderLevel: sender.Level(),
		SenderMeso:  sender.Meso(),
	})
	switch reason {
	case dueyparcel.RejectIncorrectRequest:
		reject(parcelcb.ParcelIncorrectRequestBody())
		return
	case dueyparcel.RejectMesoLimit:
		reject(parcelcb.ParcelMesoLimitBody())
		return
	case dueyparcel.RejectNotEnoughMesos:
		reject(parcelcb.ParcelNotEnoughMesosBody())
		return
	}

	// Recipient resolution (design §6.1): ByNameProvider is tenant-scoped
	// and name-filtered but not world-filtered; the channel filters to
	// s.WorldId() itself, which is also what makes the same-account check
	// below possible without a second lookup.
	candidates, err := deps.resolveRecipient(sp.RecipientName())
	if err != nil {
		l.WithError(err).Warnf("Character [%d] DUEY_ACTION SEND: unable to resolve recipient [%s].", s.CharacterId(), sp.RecipientName())
		reject(parcelcb.ParcelNameDoesNotExistBody())
		return
	}
	var recipient character.Model
	found := false
	for _, c := range candidates {
		if c.WorldId() == s.WorldId() {
			recipient = c
			found = true
			break
		}
	}
	if !found {
		l.Warnf("Character [%d] attempted to send a parcel to unknown recipient [%s].", s.CharacterId(), sp.RecipientName())
		reject(parcelcb.ParcelNameDoesNotExistBody())
		return
	}

	if recipient.AccountId() == sender.AccountId() {
		l.Warnf("Character [%d] attempted to send a parcel to their own account (recipient [%d]).", s.CharacterId(), recipient.Id())
		reject(parcelcb.ParcelSameAccountBody())
		return
	}

	pending, err := deps.countPending(recipient.Id(), s.WorldId())
	if err != nil {
		l.WithError(err).Errorf("Character [%d] DUEY_ACTION SEND: unable to check recipient [%d]'s mailbox.", s.CharacterId(), recipient.Id())
		reject(parcelcb.ParcelIncorrectRequestBody())
		return
	}
	if pending >= dueyparcel.MailboxCapacity {
		l.Warnf("Character [%d] attempted to send a parcel to recipient [%d], whose mailbox is full ([%d] pending).", s.CharacterId(), recipient.Id(), pending)
		reject(parcelcb.ParcelReceiverStorageFullBody())
		return
	}

	if sp.Quick() {
		hasTicket, terr := deps.hasTicket(s.CharacterId())
		if terr != nil {
			l.WithError(terr).Errorf("Character [%d] DUEY_ACTION SEND: unable to check for a Quick Delivery Ticket.", s.CharacterId())
			reject(parcelcb.ParcelIncorrectRequestBody())
			return
		}
		if !hasTicket {
			l.Warnf("Character [%d] attempted a quick DUEY_ACTION SEND without holding a Quick Delivery Ticket [%d].", s.CharacterId(), item.QuickDeliveryTicketId)
			reject(parcelcb.ParcelIncorrectRequestBody())
			return
		}
	}

	var sourceInventoryType byte
	var assetId uint32
	var quantity uint32
	if sp.Quantity() > 0 {
		it := inventory.Type(sp.InventoryType())
		a, ierr := deps.getItem(s.CharacterId(), it, int16(sp.Slot()))
		if ierr != nil {
			l.WithError(ierr).Warnf("Character [%d] DUEY_ACTION SEND: unable to load item in inventory [%d] slot [%d].", s.CharacterId(), it, sp.Slot())
			reject(parcelcb.ParcelIncorrectRequestBody())
			return
		}
		sourceInventoryType = byte(it)
		assetId = a.Id()
		quantity = uint32(sp.Quantity())
	}

	sg := buildParcelSendSaga(uuid.New(), uuid.New(), time.Now(), s, sender, recipient, sp.Mesos(), sp.Quick(), sp.Message(), sourceInventoryType, assetId, quantity)
	if err = deps.createSaga(sg); err != nil {
		l.WithError(err).Errorf("Character [%d] unable to initiate parcel send saga.", s.CharacterId())
	}
}

// buildParcelSendSaga assembles the parcel_send saga in design §4.3's
// order: award_mesos first (mesoAmount + fee, so an unaffordable send costs
// nothing downstream and the compensation is a credit rather than an item
// re-mint — the same reasoning note_send.go:17-19 records for
// destroy-first), then the conditional destroy_asset (the Quick Delivery
// Ticket, only when quick — FR-26, consumed on send, not on open), then
// transfer_to_parcel (composite → release_from_character + accept_to_parcel).
func buildParcelSendSaga(transactionId uuid.UUID, parcelId uuid.UUID, now time.Time, s session.Model, sender character.Model, recipient character.Model, mesoAmount uint32, quick bool, message string, sourceInventoryType byte, assetId uint32, quantity uint32) saga.Saga {
	total, _ := dueyparcel.TotalCost(mesoAmount, quick) // ValidateSend already confirmed this fits uint32
	feePaid := uint32(total) - mesoAmount

	steps := make([]saga.Step, 0, 3)

	steps = append(steps, saga.Step{
		StepId: "award_mesos",
		Status: saga.Pending,
		Action: saga.AwardMesos,
		Payload: saga.AwardMesosPayload{
			CharacterId: s.CharacterId(),
			WorldId:     s.WorldId(),
			ChannelId:   s.ChannelId(),
			ActorId:     0,
			ActorType:   "SYSTEM",
			Amount:      -int32(total),
			ShowEffect:  true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	if quick {
		steps = append(steps, saga.Step{
			StepId: "consume_quick_delivery_ticket",
			Status: saga.Pending,
			Action: saga.DestroyAsset,
			Payload: saga.DestroyAssetPayload{
				CharacterId: s.CharacterId(),
				TemplateId:  item.QuickDeliveryTicketId,
				Quantity:    1,
				RemoveAll:   false,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	steps = append(steps, saga.Step{
		StepId: "transfer_to_parcel",
		Status: saga.Pending,
		Action: saga.TransferToParcel,
		Payload: saga.TransferToParcelPayload{
			TransactionId:       transactionId,
			ParcelId:            parcelId,
			CharacterId:         s.CharacterId(),
			WorldId:             s.WorldId(),
			SourceInventoryType: sourceInventoryType,
			AssetId:             assetId,
			Quantity:            quantity,
			SenderAccountId:     sender.AccountId(),
			SenderName:          sender.Name(),
			RecipientId:         recipient.Id(),
			RecipientAccountId:  recipient.AccountId(),
			RecipientName:       recipient.Name(),
			MesoAmount:          mesoAmount,
			FeePaid:             feePaid,
			Quick:               quick,
			Message:             message,
			ReceivableAt:        now.Add(dueyReceivableDelay),
			ExpiresAt:           now.Add(dueyExpiryWindow),
		},
		CreatedAt: now,
		UpdatedAt: now,
	})

	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.ParcelSend,
		InitiatedBy:   "DUEY_ACTION_SEND",
		Steps:         steps,
	}
}
