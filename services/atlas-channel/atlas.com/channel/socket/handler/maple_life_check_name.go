package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	mlcb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound"
	msb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// mapleLifeNameValidityFunc is the seam handleMapleLifeCheckName calls
// atlas-character through, so tests can answer the Maple Life duplicate-name
// probe without a live REST round trip — the same pattern
// checkNameChangeValidityFunc (cash_shop_check_name_change.go) uses. It is a
// SEPARATE package var from that one, deliberately: this handler and the
// cash-shop rename probe answer two different ops with two different scopes
// (WORLD here, TENANT there), and a test swapping one seam must not be able
// to accidentally swap the other's behaviour too.
var mapleLifeNameValidityFunc = func(l logrus.FieldLogger, ctx context.Context, name string, worldId world.Id, scope character.NameScope) (character.NameValidityResult, error) {
	return character.NewProcessor(l, ctx).CheckNameValidity(name, worldId, scope)
}

// mapleLifeKnownReasons is the set of character.NameReason* values
// atlas-character's CheckNameValidity can return. It exists only so an
// unrecognised reason can be logged loudly (FR-3.3) rather than silently
// folded into the generic-failure arm — it is NOT a second copy of the
// reason->arm table. That table already lives in
// mlcb.MapleLifeResultRejectedBody (Task 4, libs/atlas-packet/maplelife/clientbound/result.go),
// and this handler calls it directly rather than re-deriving which arm a
// reason renders to.
var mapleLifeKnownReasons = map[string]struct{}{
	character.NameReasonDuplicate: {},
	character.NameReasonReserved:  {},
	character.NameReasonLength:    {},
	character.NameReasonRegex:     {},
}

// MapleLifeCheckNameHandleFunc handles the Maple Life (Cash/0543,
// CUICharacterSaleDlg) duplicate-name probe --
// CUICharacterSaleDlg::SendCheckDuplicateIDPacket, sent as the player types a
// candidate name into the naming dialog beginMapleLife opened.
//
// Unlike the cash-shop rename probe (cashsb.CheckNameChangeRequest, which
// shares CHECK_CHAR_NAME's opcode with login-socket character creation),
// this op has its own Maple-Life-specific opcode on every in-scope version
// with no collision found on any of them
// (libs/atlas-packet/maplelife/serverbound/check_name.go). So this is a
// standalone handler, not a branch inside CashShopCheckNameChangeHandleFunc.
func MapleLifeCheckNameHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := msb.CheckName{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		handleMapleLifeCheckName(l, ctx, wp)(s, p.Name())
	}
}

// handleMapleLifeCheckName is the shared body MapleLifeCheckNameHandleFunc
// calls. It does NOT consult the maplelife registry: that pending-dialog
// state exists to disambiguate an opcode shared with the cash-shop rename
// probe (routing outcome (B)), and this op has its own dedicated opcode on
// every in-scope version (routing outcome (A) -- see
// libs/atlas-packet/maplelife/serverbound/check_name.go), so there is
// nothing to disambiguate. A registry gate would also be actively wrong
// here: the client sends this probe WHILE the player is composing the name,
// which is before any Maple Life packet the server has seen, so a pending
// record can never be expected to exist yet.
func handleMapleLifeCheckName(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, name string) {
	return func(s session.Model, name string) {
		// NameScopeWorld, not TENANT (task-227 FR-3.2's TENANT scope is the
		// deliberately stricter rename-only case): this is a creation, so the
		// only collision that matters is within the world the character will
		// actually be created in.
		res, err := mapleLifeNameValidityFunc(l, ctx, name, s.WorldId(), character.NameScopeWorld)
		if err != nil {
			l.WithError(err).Errorf("Unable to check name validity of [%s] for account [%d].", name, s.AccountId())
			announceMapleLifeResult(l, ctx, wp, s, mlcb.MapleLifeResultBody(name, mlcb.MapleLifeResultUnknownError))
			return
		}

		if res.Valid {
			l.Debugf("Name [%s] is available for account [%d] to use for Maple Life character creation.", name, s.AccountId())
			announceMapleLifeResult(l, ctx, wp, s, mlcb.MapleLifeResultBody(name, mlcb.MapleLifeResultAvailable))
			return
		}

		if _, known := mapleLifeKnownReasons[res.Reason]; !known {
			l.Errorf("Account [%d]'s name-validity check for [%s] returned an unrecognised reason [%s]; treating as unknown error.", s.AccountId(), name, res.Reason)
		} else {
			l.Infof("Name [%s] is unavailable for account [%d]: [%s].", name, s.AccountId(), res.Reason)
		}
		announceMapleLifeResult(l, ctx, wp, s, mlcb.MapleLifeResultRejectedBody(name, res.Reason))
	}
}

// announceMapleLifeResult writes MAPLELIFE_RESULT.
func announceMapleLifeResult(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body packet.Encode) {
	if err := session.Announce(l)(ctx)(wp)(mlcb.MapleLifeResultWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to write Maple Life name-check result for account [%d].", s.AccountId())
	}
}
