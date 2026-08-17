package handler

import (
	"atlas-channel/account"
	"atlas-channel/character"
	"atlas-channel/pendingchange"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	channelworld "atlas-channel/world"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// checkPossibleTransferEligibilityIndependentFunc is the seam for the
// destination-free gate check (design's OQ-7 split), swappable in tests the
// same way the other checkPossible* funcs in this file are.
var checkPossibleTransferEligibilityIndependentFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (bool, string, error) {
	return pendingchange.NewProcessor(l, ctx).CheckTransferEligibilityIndependent(characterId)
}

// checkPossibleAccountCharactersInWorldFunc is the seam warnIfStrandingStorage
// (cash_shop_operation.go) calls through for the FR-4.7 last-character-in-
// source-world lookup, so tests can swap it the way checkPossibleAccountGetByIdFunc
// is swapped (cash_shop_check_name_change_possible.go) without a live
// atlas-character round trip. Declared here rather than beside its caller
// because this file already owns the analogous account/PIC seams this op's
// sibling BUY_WORLD_TRANSFER handler reuses.
var checkPossibleAccountCharactersInWorldFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32, worldId world.Id) ([]character.Model, error) {
	return character.NewProcessor(l, ctx).GetForAccountInWorld(accountId, worldId)
}

// checkPossibleWorldsFunc is the seam for the world-name list the ALLOWED arm
// must carry (see transferWorldNameList), swappable in tests for the same
// reason as the two seams above.
var checkPossibleWorldsFunc = func(l logrus.FieldLogger, ctx context.Context) ([]channelworld.Model, error) {
	return channelworld.NewProcessor(l, ctx).GetAll()
}

// CashShopCheckTransferWorldPossibleHandleFunc handles the standalone
// serverbound WORLD_TRANSFER op — the cash shop's "may this character change
// worlds?" request, sent when the player buys the 5401000 world-transfer item
// and the client-side CCashShop::CheckTransferWorldPossible gate passes. It is
// NOT an arm of CashShopOperationHandle: the request has its own opcode and no
// leading mode byte, so it is registered by name in main.go like any other
// standalone handler.
//
// The body carries characterId (absent on jms_v185 — that region identifies
// the character from session state, see
// cashsb.CheckTransferWorldPossible.CharacterId's doc comment; this handler
// falls back to s.CharacterId() when the wire field is not present) and the
// account's second-password credential (an 8-digit birthday code pre-v95, a
// string SPW on v95+/jms_v185 — cashsb.TransferCredentialIsString).
//
// This op does NOT carry a destination world — BUY_WORLD_TRANSFER supplies
// that later — so the destination-dependent gates of atlas-character's
// transfer-eligibility endpoint (world_same, world_unknown/world_full,
// no_character_slot, name_taken) cannot be evaluated here and remain
// BUY-time only, via pendingchange.RequestWorldTransfer's full gate table
// (wired in the BUY_WORLD_TRANSFER handler).
//
// The remaining destination-independent gates (is_gm, banned,
// is_guild_master, in_family, trade_open, merchant_open, mts_listings_open)
// ARE evaluated here, via atlas-character's destination-free
// GET .../transfer-eligibility-independent route
// (pending_change.CheckTransferEligibilityIndependent) — this closes OQ-7
// (docs/tasks/task-227-cash-name-change-world-transfer/bug-world-transfer-eligibility-reasons.md,
// "The better fix for 2c"). A rejection from that check answers via
// cashcb.CheckTransferWorldPossibleResultRejectedBody, which routes in_family
// to its own confirmed client arm (StringPool 5017) rather than collapsing to
// the generic UNKNOWN_ERROR every other rejection on this op still uses. The
// real per-purchase gates (both halves) still run again when the
// pending-change record is created (pendingchange.RequestWorldTransfer,
// wired in the BUY_WORLD_TRANSFER handler) — that second, authoritative
// evaluation is unchanged; this one is advisory, so the player is told before
// the license notice rather than after picking a destination.
//
// The ALLOWED arm MUST carry a non-empty world-name list. An empty list is
// not a cosmetic gap — it crashes the v83 client. Chain, from
// MapleStory_dump.exe (GMS v83), recorded in
// docs/tasks/task-227-cash-name-change-world-transfer/bug-world-transfer-client-crash.md:
// CCashShop::OnCheckTransferWorldPossibleResult @0x47bd9b arm 0 opens
// CUITransferWorldLicenseNotice; its OK button (@0x7ef6e3, a2 == 1) DoModals
// CUITransferWorldSelectDlg, whose OnCreate @0x7efc22 guards the combo fill
// loop against an empty m_asWorldName but then calls
// CCtrlComboBox::SetSelected(0) (@0x4c738b) unguarded — which walks into
// sub_4C7379 @0x4c7379, dereferencing the null head of an empty ZList at
// [0x00000004]. So the handler answers UNKNOWN_ERROR rather than ALLOWED when
// the list cannot be produced: a refusal arm renders a notice and returns the
// player to the Cash Shop, where an ALLOWED-with-no-list kills the client.
func CashShopCheckTransferWorldPossibleHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CheckTransferWorldPossible{}
		p.Decode(l, ctx)(r, readerOptions)
		// p.String() REDACTS the credential the body carries (the account
		// second password / birthday code). Never log p.Spw() or
		// p.BirthDate().
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		characterId := p.CharacterId()
		if !cashsb.TransferBodyHasCharacterId(ctx) {
			characterId = s.CharacterId()
		}

		a, err := checkPossibleAccountGetByIdFunc(l, ctx, s.AccountId())
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve account [%d] for world-transfer credential validation.", s.AccountId())
			announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultBody(characterId, cashcb.CheckTransferWorldPossibleUnknownError, 0, nil))
			return
		}

		ipAddress := remoteIpAddress(s)

		if !transferWorldCredentialMatches(ctx, p, a) {
			l.Debugf("Incorrect world-transfer credential for account [%d].", s.AccountId())
			_, _, rErr := checkPossibleRecordPicAttemptFunc(l, ctx, s.AccountId(), false, ipAddress)
			if rErr != nil {
				l.WithError(rErr).Errorf("Unable to record PIC attempt for account [%d].", s.AccountId())
			}
			// Neither a bare credential mismatch nor a tripped lockout has a
			// dedicated arm on this op (only IN_FAMILY, arm 8, has
			// independently confirmed text — see the result codec's doc
			// comment); UNKNOWN_ERROR is the existing rejection path reused
			// for both, per task-227 Task 26 ruling 4.
			announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultBody(characterId, cashcb.CheckTransferWorldPossibleUnknownError, 0, nil))
			return
		}

		if _, _, rErr := checkPossibleRecordPicAttemptFunc(l, ctx, s.AccountId(), true, ipAddress); rErr != nil {
			l.WithError(rErr).Errorf("Unable to record PIC attempt for account [%d].", s.AccountId())
		}

		// Destination-independent gate check (design's OQ-7 split, see the
		// type-level doc comment above). An infrastructure failure here
		// refuses the transfer rather than risking a false ALLOWED — the
		// same fail-closed posture the world-list lookup below already uses
		// — and reuses UNKNOWN_ERROR since no dedicated arm exists for "the
		// check itself could not run".
		eligible, reason, eErr := checkPossibleTransferEligibilityIndependentFunc(l, ctx, characterId)
		if eErr != nil {
			l.WithError(eErr).Errorf("Unable to check destination-independent transfer eligibility for character [%d]; refusing the world-transfer check.", characterId)
			announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultBody(characterId, cashcb.CheckTransferWorldPossibleUnknownError, 0, nil))
			return
		}
		if !eligible {
			l.Infof("World-transfer check for character [%d] rejected by a destination-independent gate: %s.", characterId, reason)
			announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultRejectedBody(characterId, reason, nil))
			return
		}

		ws, wErr := checkPossibleWorldsFunc(l, ctx)
		if wErr != nil {
			l.WithError(wErr).Errorf("Unable to retrieve the world list for character [%d]; refusing the world-transfer check rather than sending an empty list.", characterId)
			announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultBody(characterId, cashcb.CheckTransferWorldPossibleUnknownError, 0, nil))
			return
		}
		worldNames := transferWorldNameList(l, ws)
		if len(worldNames) == 0 {
			l.Errorf("The world list is empty; refusing the world-transfer check for character [%d] rather than sending an empty list.", characterId)
			announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultBody(characterId, cashcb.CheckTransferWorldPossibleUnknownError, 0, nil))
			return
		}

		announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultAllowedBody(characterId, a.BirthDate(), worldNames))

		// FR-4.7's storage-stranding courtesy warning is NOT emitted here.
		// It used to be, immediately after the ALLOWED result above, but that
		// collided with the client's license-notice modal: ALLOWED opens
		// CUITransferWorldLicenseNotice via DoModal, and the warning's own
		// POP_UP world message opens a SECOND modal (CUtilDlg::Notice) inside
		// that one's nested message loop, which steals the input grab and
		// makes the license notice unresponsive. See
		// docs/tasks/task-227-cash-name-change-world-transfer/bug-world-transfer-eligibility-reasons.md
		// (Symptom 1) for the full IDA trace, and its Ruling 1: the warning
		// now fires from handleBuyWorldTransfer (cash_shop_operation.go),
		// once the player has dismissed that dialog and picked a
		// destination.
	}
}

// transferWorldNameList renders the world set as the wire list the client
// indexes BY WORLD ID, not by position in the server's result set.
//
// The client never sends a world id back. CUITransferWorldSelectDlg::OnCreate
// (@0x7efc22) fills its combo from CCashShop::m_asWorldName in list order;
// sub_7F00D2 @0x7f00d2 stores the combo's m_nSelected (the raw INDEX) into
// m_nResult; GetResult @0x7f00e2 returns it verbatim; and
// CUITransferWorldLicenseNotice::OnButtonClicked @0x7ef6e3 hands that index to
// CCashShop::SendBuyTransferWorldItemPacket as nTargetWorld. The
// BUY_WORLD_TRANSFER handler (cash_shop_operation.go, handleBuyWorldTransfer)
// then reads that field as world.Id(sp.TargetWorld()).
//
// So index i of this slice MUST be world i's name, or a transfer silently
// lands in the wrong world. The list is therefore sized to max(world id)+1 and
// filled by id, not appended in result order. A gap (an id with no world)
// leaves a blank combo entry; selecting it resolves to a world atlas-character
// does not know, which pendingchange.RequestWorldTransfer already rejects with
// world_unknown. That is the honest, safe rendering — collapsing the gap would
// shift every later world's index and misroute the purchase.
func transferWorldNameList(l logrus.FieldLogger, ws []channelworld.Model) []string {
	if len(ws) == 0 {
		return nil
	}

	maxId := ws[0].Id()
	for _, w := range ws {
		if w.Id() > maxId {
			maxId = w.Id()
		}
	}

	names := make([]string, int(maxId)+1)
	for _, w := range ws {
		names[int(w.Id())] = w.Name()
	}
	for i, n := range names {
		if n == "" {
			l.Warnf("No world [%d] exists; the cash-shop world-transfer list carries a blank entry at that index so index and world id stay aligned.", i)
		}
	}
	return names
}

// transferWorldCredentialMatches mirrors nameChangeCredentialMatches (see
// cash_shop_check_name_change_possible.go) for the WORLD_TRANSFER op's own
// version gate, cashsb.TransferCredentialIsString — which, unlike the
// name-change gate, also covers jms_v185 (task-227 Task 26 ruling 2's JMS
// arm).
func transferWorldCredentialMatches(ctx context.Context, p cashsb.CheckTransferWorldPossible, a account.Model) bool {
	if cashsb.TransferCredentialIsString(ctx) {
		return p.Spw() == a.PIC()
	}
	if a.BirthDate() == 0 {
		return false
	}
	return p.BirthDate() == a.BirthDate()
}

// announceTransferWorldPossible writes
// CASHSHOP_CHECK_TRANSFER_WORLD_POSSIBLE_RESULT.
func announceTransferWorldPossible(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body packet.Encode) {
	if err := session.Announce(l)(ctx)(wp)(cashcb.CashShopCheckTransferWorldPossibleResultWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to write world-transfer-possible result for character [%d].", s.CharacterId())
	}
}
