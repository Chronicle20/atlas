package handler

import (
	"atlas-channel/cashshop/inventory/compartment"
	"atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

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

// findGiftAsset locates the cash-shop asset the gift-forward NOTE_ACTION SEND
// packet references by its cash-item serial number (GiftSN). Returns
// (giftFrom, true) when found; ("", false) when not.
func findGiftAsset(cp compartment.Model, giftSN uint64) (string, bool) {
	for _, as := range cp.Assets() {
		if uint64(as.Item().CashId()) == giftSN {
			return as.GiftFrom(), true
		}
	}
	return "", false
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
func handleNoteGiftForward(l logrus.FieldLogger, ctx context.Context) func(s session.Model, sp *notesb.OperationSend) {
	return func(s session.Model, sp *notesb.OperationSend) {
		// TODO select correct compartment (cash_shop_entry.go:74 applies here
		// identically).
		cp, err := compartment.NewProcessor(l, ctx).GetByAccountIdAndType(s.AccountId(), compartment.TypeExplorer)
		if err != nil {
			l.WithError(err).Warnf("Character [%d] NOTE_ACTION SEND gift-forward: unable to load cash compartment. Not creating note.", s.CharacterId())
			return
		}

		giftFrom, found := findGiftAsset(cp, sp.GiftSN())
		if !found {
			l.Warnf("Character [%d] NOTE_ACTION SEND gift-forward: no cash-shop asset with SN [%d]. Not creating note.", s.CharacterId(), sp.GiftSN())
			return
		}
		if giftFrom == "" || giftFrom != sp.ToName() {
			l.Warnf("Character [%d] NOTE_ACTION SEND gift-forward: asset SN [%d] giftFrom [%s] does not match toName [%s]. Not creating note.", s.CharacterId(), sp.GiftSN(), giftFrom, sp.ToName())
			return
		}

		gifter, err := character.NewProcessor(l, ctx).GetByName(sp.ToName())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] NOTE_ACTION SEND gift-forward: unable to resolve gifter [%s]. Not creating note.", s.CharacterId(), sp.ToName())
			return
		}

		sg := buildGiftForwardSaga(uuid.New(), time.Now(), s.CharacterId(), gifter.Id(), sp.Message())
		if err = noteGiftForwardSagaCreateFunc(l, ctx, sg); err != nil {
			l.WithError(err).Errorf("Character [%d] NOTE_ACTION SEND gift-forward: unable to create note_send saga.", s.CharacterId())
		}
	}
}
