package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	cashsb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// checkNameChangeValidityFunc is the seam the handler calls atlas-character
// through, so tests can answer the name-validity probe without a live REST
// round trip — the same pattern
// cash_shop_check_name_change_possible.go uses for the account lookup.
// scope is a parameter rather than a constant baked into this default
// implementation so the handler's FR-3.2 choice is observable through the
// seam — a test can assert the handler asked for TENANT scope instead of
// re-stating the constant on the far side of the swap and proving nothing.
var checkNameChangeValidityFunc = func(l logrus.FieldLogger, ctx context.Context, name string, worldId world.Id, scope character.NameScope) (character.NameValidityResult, error) {
	return character.NewProcessor(l, ctx).CheckNameValidity(name, worldId, scope)
}

// CashShopCheckNameChangeHandleFunc handles the cash shop's candidate-name
// probe — CCashShop::SendCheckDuplicateIDPacket, sent as the player types a new
// name into the rename dialog after buying the 5400000 name-change item.
//
// This op shares its opcode with character creation's name check; the login
// socket binds that opcode to CharacterCheckNameHandle and answers with
// CHARACTER_NAME_RESPONSE, while the channel socket binds it to this handler
// and answers with CASHSHOP_CHECK_NAME_CHANGE. See
// cashsb.CheckNameChangeRequest's doc comment for the evidence and for why the two
// halves need distinct handler names in the tenant templates.
//
// The body carries the candidate name and nothing else — no credential. The
// credential gate for a rename lives on the separate
// CASHSHOP_CHECK_NAME_CHANGE_POSSIBLE op (cash_shop_check_name_change_possible.go),
// which the client sends first; this probe is purely "is this string usable".
// So there is nothing to authenticate here, and nothing sensitive to redact.
//
// The client's own switch is a THREE-way signed branch — available / taken /
// unknown error — which cannot express the four-reason taxonomy atlas-character
// returns. cashcb.CheckNameChangeRejectedBody owns that lossy mapping (only
// name_taken has a distinct arm); see its doc comment. Do not add reason
// strings here.
func CashShopCheckNameChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := cashsb.CheckNameChangeRequest{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		// NameScopeTenant, not WORLD (task-227 FR-3.2): a rename must not
		// produce a name that already exists in ANY world of the tenant,
		// because a later world transfer would then collide.
		res, err := checkNameChangeValidityFunc(l, ctx, p.Name(), s.WorldId(), character.NameScopeTenant)
		if err != nil {
			l.WithError(err).Errorf("Unable to check name validity of [%s] for character [%d].", p.Name(), s.CharacterId())
			announceCheckNameChange(l, ctx, wp, s, cashcb.CheckNameChangeResultBody(p.Name(), cashcb.CheckNameChangeUnknownError))
			return
		}

		if res.Valid {
			l.Debugf("Name [%s] is available for character [%d] to rename to.", p.Name(), s.CharacterId())
			announceCheckNameChange(l, ctx, wp, s, cashcb.CheckNameChangeAvailableBody(p.Name()))
			return
		}

		reason := nameChangeRejectionReason(res.Reason)
		l.Infof("Name [%s] is unavailable for character [%d]: [%s].", p.Name(), s.CharacterId(), reason)
		announceCheckNameChange(l, ctx, wp, s, cashcb.CheckNameChangeRejectedBody(p.Name(), reason))
	}
}

// nameChangeRejectionReasons translates atlas-character's name-validity reason
// onto design §6's closed taxonomy, which is what
// cashcb.CheckNameChangeRejectedBody consumes. It is a map rather than a switch
// so a test can assert every reason atlas-character can return is covered
// instead of trusting the branches to have listed them.
var nameChangeRejectionReasons = map[string]string{
	character.NameReasonDuplicate: "name_taken",
	character.NameReasonReserved:  "name_reserved",
	character.NameReasonLength:    "name_invalid_length",
	character.NameReasonRegex:     "name_invalid_charset",
}

// nameChangeRejectionReason maps an unrecognised reason onto name_reserved
// rather than name_taken. Both render through the client as a refusal, but
// name_taken is the one arm that renders the SPECIFIC "this name is currently
// in use" string — claiming that for a reason we do not recognise would tell
// the player something we have not established. name_reserved falls through to
// the client's generic error arm, which is the honest answer.
func nameChangeRejectionReason(reason string) string {
	if mapped, ok := nameChangeRejectionReasons[reason]; ok {
		return mapped
	}
	return "name_reserved"
}

// announceCheckNameChange writes CASHSHOP_CHECK_NAME_CHANGE.
func announceCheckNameChange(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model, body packet.Encode) {
	if err := session.Announce(l)(ctx)(wp)(cashcb.CashShopCheckNameChangeWriter)(body)(s); err != nil {
		l.WithError(err).Errorf("Unable to write name-change availability result for character [%d].", s.CharacterId())
	}
}
