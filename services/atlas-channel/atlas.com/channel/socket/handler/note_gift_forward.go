package handler

import (
	"atlas-channel/cashshop"
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	notesb "github.com/Chronicle20/atlas/libs/atlas-packet/note/serverbound"
)

// noteGiftForwardSagaCreateFunc is a test seam for saga creation (precedent:
// npcItemUseSagaCreateFunc, npc_item_use.go).
var noteGiftForwardSagaCreateFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// buildGiftForwardSaga assembles the single-step note_send saga for a gift
// acknowledgement: no destroy step, because the note is paid for by the gift
// purchase (design §2.2), not by a Note item. senderId/receiverId here are
// the note's sender/receiver — the recipient of the gift (who typed the
// message) sends the note, addressed to the character who gifted the item.
func buildGiftForwardSaga(transactionId uuid.UUID, now time.Time, senderId uint32, receiverId uint32, message string) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.NoteSend,
		InitiatedBy:   "NOTE_ACTION_GIFT_FORWARD",
		Steps: []saga.Step{
			{
				StepId: "create_note",
				Status: saga.Pending,
				Action: saga.CreateNote,
				Payload: saga.CreateNotePayload{
					SenderId:   senderId,
					ReceiverId: receiverId,
					Message:    message,
					// Flag 0 = plain note; see note_send.go's buildNoteSendSaga
					// comment for why non-zero flags are reserved for other
					// memo render templates.
					Flag: 0,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// buildGiftFameSaga assembles a single-step award_fame saga that fames the
// gifter by +1 when their gift is acknowledged. This is deliberately a
// separate InventoryTransaction saga from the note_send saga
// (buildGiftForwardSaga), not a second step on it: compensateNoteSend
// terminates the whole saga on any step failure and emits a
// StatusEventTypeFailed carrying the sender's characterId, which the channel
// turns into a MEMO_RESULT SEND_ERROR announce. The client has already shown
// SP_2713 "The note has successfully been sent." unconditionally before any
// server reply, so a fame failure must not be able to drag the note saga
// into that error path. A standalone InventoryTransaction saga carrying one
// award_fame step is the established precedent for exactly this — see
// atlas-notes' buildFameAwardSaga (note/processor.go).
func buildGiftFameSaga(transactionId uuid.UUID, now time.Time, gifterId uint32, worldId world.Id, channelId channel.Id) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.InventoryTransaction,
		InitiatedBy:   "NOTE_ACTION_GIFT_FORWARD_FAME",
		Steps: []saga.Step{
			{
				StepId: "award_fame",
				Status: saga.Pending,
				Action: saga.AwardFame,
				Payload: saga.AwardFamePayload{
					CharacterId: gifterId,
					WorldId:     worldId,
					ChannelId:   channelId,
					Amount:      1,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// findGiftAsset locates the cash-shop asset the gift-forward NOTE_ACTION SEND
// packet references by its cash-item serial number (GiftSN). Returns the
// asset's giftFrom and giftNoteSent (task-240 Defect I) when found, with
// found == true; ("", false, false) when not.
func findGiftAsset(cp compartment.Model, giftSN uint64) (giftFrom string, giftNoteSent bool, found bool) {
	for _, as := range cp.Assets() {
		if uint64(as.Item().CashId()) == giftSN {
			return as.GiftFrom(), as.GiftNoteSent(), true
		}
	}
	return "", false, false
}

// noteGiftForwardMarkSentFunc is a test seam for the MARK_GIFT_NOTE_SENT
// command (precedent: noteGiftForwardSagaCreateFunc above).
var noteGiftForwardMarkSentFunc = func(l logrus.FieldLogger, ctx context.Context, s session.Model, cashId int64) error {
	return cashshop.NewProcessor(l, ctx).MarkGiftNoteSent(s.AccountId(), s.CharacterId(), cashId)
}

// noteGiftForwardCompartmentFunc is a test seam for the sender's cash-shop
// compartment lookup (precedent: noteGiftForwardSagaCreateFunc above) -- lets
// tests exercise the real gates in handleNoteGiftForward against a
// builder-constructed compartment.Model, with no HTTP involved.
var noteGiftForwardCompartmentFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32) (compartment.Model, error) {
	return compartment.NewProcessor(l, ctx).GetByAccountIdAndType(accountId, compartment.TypeExplorer)
}

// noteGiftForwardCharacterFunc is a test seam for gifter resolution
// (precedent: noteGiftForwardSagaCreateFunc above).
var noteGiftForwardCharacterFunc = func(l logrus.FieldLogger, ctx context.Context, name string) (character.Model, error) {
	return character.NewProcessor(l, ctx).GetByName(name)
}

// handleNoteGiftForward implements the gift-forward branch of NOTE_ACTION
// SEND (giftFlag == 1) — the only branch a legitimate v83+ client writes
// (CCashShop::OnCashItemResLoadGiftDone). It does not consume anything and
// does not route through handleNoteSendRequest's consume-gated path: the
// note is paid for by the gift purchase, not by a Note item.
//
// The anti-tamper gate that replaces the Note-item ownership check: a client
// can only mint a free note addressed to a character who actually gifted it
// an item it actually holds. Both the asset lookup (by GiftSN) and the
// GiftFrom == ToName match must succeed, or nothing is created. No failure
// here announces anything to the client — the client has already shown
// SP_2713 "The note has successfully been sent." unconditionally, before any
// server reply, so there is no arm to answer on.
//
// A second gate, independent of the above, closes task-240 Defect I: the
// asset's GiftNoteSent flag must not already be set. GiftAcknowledged is
// deliberately NOT consulted here — it drains on the LOAD_GIFT_SUCCESS
// announce, before this packet can ever arrive, so gating on it would reject
// every legitimate note (see "### Interaction with Defect G" in the bug
// writeup). Known limitation, not fixed here: MarkGiftNoteSent is an
// asynchronous Kafka round trip, so two acknowledgement packets racing
// inside that window can both pass this gate before either write lands.
// This narrows the exposure from unbounded to a single race.
func handleNoteGiftForward(l logrus.FieldLogger, ctx context.Context) func(s session.Model, sp *notesb.OperationSend) {
	return func(s session.Model, sp *notesb.OperationSend) {
		// TODO select correct compartment (cash_shop_entry.go:74 applies here
		// identically).
		cp, err := noteGiftForwardCompartmentFunc(l, ctx, s.AccountId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] NOTE_ACTION SEND gift-forward: unable to load cash compartment. Not creating note.", s.CharacterId())
			return
		}

		giftFrom, giftNoteSent, found := findGiftAsset(cp, sp.GiftSN())
		if !found {
			l.Warnf("Character [%d] NOTE_ACTION SEND gift-forward: no cash-shop asset with SN [%d]. Not creating note.", s.CharacterId(), sp.GiftSN())
			return
		}
		if giftFrom == "" || giftFrom != sp.ToName() {
			l.Warnf("Character [%d] NOTE_ACTION SEND gift-forward: asset SN [%d] giftFrom [%s] does not match toName [%s]. Not creating note.", s.CharacterId(), sp.GiftSN(), giftFrom, sp.ToName())
			return
		}
		if giftNoteSent {
			l.Warnf("Character [%d] NOTE_ACTION SEND gift-forward: asset SN [%d] has already had its note sent. Not creating a second note.", s.CharacterId(), sp.GiftSN())
			return
		}

		gifter, err := noteGiftForwardCharacterFunc(l, ctx, sp.ToName())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] NOTE_ACTION SEND gift-forward: unable to resolve gifter [%s]. Not creating note.", s.CharacterId(), sp.ToName())
			return
		}

		sg := buildGiftForwardSaga(uuid.New(), time.Now(), s.CharacterId(), gifter.Id(), sp.Message())
		if err = noteGiftForwardSagaCreateFunc(l, ctx, sg); err != nil {
			l.WithError(err).Errorf("Character [%d] NOTE_ACTION SEND gift-forward: unable to create note_send saga.", s.CharacterId())
			return
		}

		if gifter.Id() == s.CharacterId() {
			l.Debugf("Character [%d] NOTE_ACTION SEND gift-forward: self-gift, skipping fame award.", s.CharacterId())
		} else {
			fg := buildGiftFameSaga(uuid.New(), time.Now(), gifter.Id(), s.WorldId(), s.ChannelId())
			if err = noteGiftForwardSagaCreateFunc(l, ctx, fg); err != nil {
				l.WithError(err).Errorf("Character [%d] NOTE_ACTION SEND gift-forward: unable to create award_fame saga for gifter [%d].", s.CharacterId(), gifter.Id())
			}
		}

		if err = noteGiftForwardMarkSentFunc(l, ctx, s, int64(sp.GiftSN())); err != nil {
			l.WithError(err).Errorf("Character [%d] NOTE_ACTION SEND gift-forward: unable to mark gift note sent for SN [%d].", s.CharacterId(), sp.GiftSN())
		}
	}
}
