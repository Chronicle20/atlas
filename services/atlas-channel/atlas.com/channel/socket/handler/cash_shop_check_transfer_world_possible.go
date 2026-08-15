package handler

import (
	"atlas-channel/account"
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// checkPossibleAccountCharactersInWorldFunc is the seam
// CashShopCheckTransferWorldPossibleHandleFunc calls through for the FR-4.7
// last-character-in-source-world lookup, so tests can swap it the way
// checkPossibleAccountGetByIdFunc is swapped (cash_shop_check_name_change_possible.go)
// without a live atlas-character round trip.
var checkPossibleAccountCharactersInWorldFunc = func(l logrus.FieldLogger, ctx context.Context, accountId uint32, worldId world.Id) ([]character.Model, error) {
	return character.NewProcessor(l, ctx).GetForAccountInWorld(accountId, worldId)
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
// no_character_slot, name_taken) cannot be evaluated here, and that endpoint
// has no destination-agnostic form (services/atlas-character/atlas.com/character/pending_change/resource.go
// handleGetTransferEligibility requires destinationWorldId on every call, and
// pending_change.CheckTransferEligibility's own gate 1 compares it against
// the character's current world). The remaining destination-independent gates
// (is_gm, banned, is_guild_master, in_family, trade_open, merchant_open,
// mts_listings_open) are evaluated by the SAME endpoint's SAME gate table, so
// they cannot be split out without inventing a second entry point on
// atlas-character that this task's brief does not authorize. Per task-227
// Task 26 ruling 5, this is reported as a genuine design gap rather than
// invented: this handler validates the credential and the PIC-attempt
// lockout only, exactly as the sibling name-change handler does, and answers
// ALLOWED on a valid credential with no further gate evaluation. The real
// per-purchase eligibility gates already run when the pending-change record
// is created (pendingchange.RequestWorldTransfer, wired in Task 25's
// BUY_WORLD_TRANSFER handler), which is the first point a destination world
// is known.
//
// The world-name list (cashcb.CheckTransferWorldPossibleResult.WorldNames) is
// likewise left empty here: atlas-channel's world package
// (services/atlas-channel/atlas.com/channel/world) has no "list all worlds"
// lookup today, only GetById(worldId). Populating the list needs a new
// atlas-world REST client this task's brief does not list either.
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

		announceTransferWorldPossible(l, ctx, wp, s, cashcb.CheckTransferWorldPossibleResultAllowedBody(characterId, a.BirthDate(), nil))

		// FR-4.7: warn (pink text) when this transfer would strand the
		// account's shared-per-world storage. Emitted AFTER the result
		// packet, deliberately: the credential check the client is waiting
		// on has already been answered above, so a slow/failed
		// last-character lookup below can never delay or block that
		// response — see warnIfStrandingStorage's own fail-open contract.
		warnIfStrandingStorage(l, ctx, wp, s, characterId)
	}
}

// warnIfStrandingStorage implements task-227 Task 26 ruling from the FR-4.7
// fix-round-2 brief: storage is keyed (tenant, world, account) and shared by
// every character the account owns in that world (FR-4.6), so it is only
// stranded when the transferring character is the account's LAST character
// in the SOURCE world (s.WorldId() — this op carries no destination world,
// see the handler's own doc comment). This is advisory, not a gate, so it
// FAILS OPEN: a lookup error is logged and swallowed, never surfaced to the
// player and never allowed to affect the (already-announced) check result.
func warnIfStrandingStorage(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, characterId uint32) {
	chars, err := checkPossibleAccountCharactersInWorldFunc(l, ctx, s.AccountId(), s.WorldId())
	if err != nil {
		l.WithError(err).Errorf("Unable to determine whether world transfer strands storage for account [%d] world [%d]; skipping the courtesy warning.", s.AccountId(), s.WorldId())
		return
	}

	isLast := len(chars) == 1 && chars[0].Id() == characterId
	if !isLast {
		return
	}

	msg := "Your Cash Shop storage in this world is tied to your account, not this character. Because this is your only remaining character here, it will become inaccessible once the transfer completes."
	// medal="" and characterName="" render this as a system notice, not a
	// player megaphone -- the same convention every other system pink-text
	// caller uses (e.g. system_message/consumer.go, saga/consumer.go).
	if err := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePinkTextBody("", "", msg))(s); err != nil {
		l.WithError(err).Errorf("Unable to write storage-stranding warning for character [%d].", s.CharacterId())
	}
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
