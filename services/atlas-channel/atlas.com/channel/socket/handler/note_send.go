package handler

import (
	character2 "atlas-channel/character"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	notecb "github.com/Chronicle20/atlas/libs/atlas-packet/note/clientbound"
)

// buildNoteSendSaga assembles the note_send saga: destroy-first (FR-5) —
// the Note item is confirmed consumed before the note exists; if note
// creation then fails, the orchestrator re-awards the item.
func buildNoteSendSaga(transactionId uuid.UUID, now time.Time, senderId uint32, templateId uint32, receiverId uint32, message string) saga.Saga {
	return saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.NoteSend,
		InitiatedBy:   "NOTE_SEND",
		Steps: []saga.Step{
			{
				StepId: "consume_note_item",
				Status: saga.Pending,
				Action: saga.DestroyAsset,
				Payload: saga.DestroyAssetPayload{
					CharacterId: senderId,
					TemplateId:  templateId,
					Quantity:    1,
					RemoveAll:   false,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				StepId: "create_note",
				Status: saga.Pending,
				Action: saga.CreateNote,
				Payload: saga.CreateNotePayload{
					SenderId:   senderId,
					ReceiverId: receiverId,
					Message:    message,
					// Flag 0 = plain note -- sender + message only, no extra
					// block. The client's memo renderer (CMemoListDlg::DrawMemo,
					// v83 sub_64B1A5@0x64b1a5) reserves 1 for the gift-delivered
					// + fame-gained notice (StringPool 3366/3367, note_gift_forward.go),
					// 2 for gift-delivered without the fame line, and 3 for a
					// wedding invitation (discardSpecialFlag). A player-typed
					// note is none of those, so it must stay 0. See
					// re-memo-nflag.md (§"Resolved") for the full decode and
					// per-version sweep.
					Flag: 0,
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

// handleNoteSendRequest runs the shared pre-flight checks for both note send
// paths (USE_CASH_ITEM note arm and NOTE_ACTION SEND) and, if they pass,
// creates the note_send saga. Pre-flight rejections announce MEMO_RESULT
// SEND_ERROR inline and consume nothing (FR-7).
func handleNoteSendRequest(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, templateId uint32, toName string, message string) {
	return func(s session.Model, templateId uint32, toName string, message string) {
		tc, err := character2.NewProcessor(l, ctx).GetByName(toName)
		if err != nil {
			l.WithError(err).Warnf("Character [%d] attempted to send a note to unknown receiver [%s].", s.CharacterId(), toName)
			_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorReceiverUnknown))(s)
			return
		}

		// Receiver-online check (design §4.1 step 4). Scope: the session
		// registry only tracks THIS channel's sessions; a receiver online on
		// another channel is not detected and the note is stored normally —
		// documented limitation, no cross-channel lookup exists in
		// atlas-channel today.
		if _, oerr := session.NewProcessor(l, ctx).GetByCharacterId(s.Field().Channel())(tc.Id()); oerr == nil {
			l.Debugf("Character [%d] attempted to send a note to online receiver [%d].", s.CharacterId(), tc.Id())
			_ = session.Announce(l)(ctx)(wp)(notecb.NoteOperationWriter)(notecb.NoteSendErrorBody(notecb.NoteSendErrorReceiverOnline))(s)
			return
		}

		ns := buildNoteSendSaga(uuid.New(), time.Now(), s.CharacterId(), templateId, tc.Id(), message)
		if err = saga.NewProcessor(l, ctx).Create(ns); err != nil {
			l.WithError(err).Errorf("Character [%d] unable to initiate note send saga.", s.CharacterId())
		}
	}
}
