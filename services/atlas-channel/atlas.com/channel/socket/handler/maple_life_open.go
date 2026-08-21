package handler

import (
	"atlas-channel/maplelife"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// beginMapleLife records the pending Maple Life character-creation dialog
// for the calling account, offered but not yet submitted (maplelife.PhaseOpen).
// It writes no packet: the client opened its own CUICharacterSaleDlg the
// instant it sent the classification-543 sub-body (task-246 design §3), so
// there is nothing left for the server to render here.
//
// Account and world come from the session, never the packet (FR-4.2) -- the
// session is the only trustworthy source of "which account is this" for an
// entry point that exists specifically because the account has no character
// yet to authenticate through.
//
// A second call for the same account (double-click, dropped ack, or a retry)
// replaces rather than duplicates the pending entry (Registry.Put, design
// §3): the newest Open always wins, which is what
// TestBeginMapleLifeIsIdempotent asserts.
//
// Ownership of the source slot is already enforced by
// CharacterCashItemUseHandleFunc's cashItemInSlotFunc check against the
// common ItemUse prefix before any arm -- including this one -- is reached
// (character_cash_item_use.go:61-66, FR-5.3). beginMapleLife does not
// re-check it.
func beginMapleLife(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, itemId item.Id, source slot.Position, updateTime uint32) {
	return func(s session.Model, itemId item.Id, source slot.Position, updateTime uint32) {
		t := tenant.MustFromContext(ctx)
		maplelife.GetRegistry().Put(t, s.AccountId(), maplelife.Entry{
			CharacterId: s.CharacterId(),
			WorldId:     s.WorldId(),
			ItemId:      itemId,
			Slot:        source,
			UpdateTime:  updateTime,
			Phase:       maplelife.PhaseOpen,
			At:          time.Now(),
		})
		l.Debugf("Account [%d] opened the Maple Life dialog with item [%d] in slot [%d].", s.AccountId(), itemId, source)
	}
}
