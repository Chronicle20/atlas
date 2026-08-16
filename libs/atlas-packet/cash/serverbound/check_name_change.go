package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CashShopCheckNameChangeHandle = "CashShopCheckNameChangeHandle"

// CheckNameChangeRequest is the serverbound request behind
// CASHSHOP_CHECK_NAME_CHANGE — the cash shop's "is this candidate name
// available?" probe, sent while the player composes a rename in the
// CUIChangingCharacterName dialog. Its answer is
// cash/clientbound.CheckNameChange (CCashShop::OnCheckDuplicatedIDResult).
//
// # This op shares its opcode with character creation's name check
//
// The client has ONE serverbound opcode for "check this character name" —
// docs/packets/MapleStory Ops - ServerBound.csv row CHECK_CHAR_NAME lists two
// senders against a single opcode column:
//
//	CLogin::SendCheckDuplicateIDPacket      (character creation, login socket)
//	CCashShop::SendCheckDuplicateIDPacket   (cash shop rename, channel socket)
//
// The two senders are distinguished by WHICH SOCKET the request arrives on,
// not by any wire field. atlas-login binds the opcode to
// character/serverbound.CharacterCheckNameHandle; atlas-channel binds the
// SAME opcode to this handler. That is why the tenant templates carry two
// handler entries at one opCode with disjoint `services` — a legal shape
// (services/atlas-configurations/.../socket/validate.go only rejects the same
// NAME twice at one opcode) and the reason this type exists at all rather
// than the channel reusing charsb.CheckName: the handler NAME is the binding
// key, and the two sockets answer with different clientbound ops
// (CHARACTER_NAME_RESPONSE vs CASHSHOP_CHECK_NAME_CHANGE).
//
// Body, identical on every applicable version — v48, v61, v72, v79, v83, v84,
// v87, v92, v95:
//
//	EncodeStr  sCharName
//
// Evidence: docs/packets/ida-exports/gms_v*.json records
// CLogin::SendCheckDuplicateIDPacket as a single `EncodeStr` (name) on v61,
// v72, v79, v83, v84, v87, v95 — the one op both senders build. The receiving
// side corroborates the shape: CCashShop::OnCheckDuplicatedIDResult and
// CLogin::OnCheckDuplicatedIDResult decode the SAME body (DecodeStr + Decode1)
// on all nine versions, i.e. the cash-shop and login halves of this exchange
// are the same packets throughout. The CCashShop sender itself is not named in
// any checked-in IDB export, so the cash-shop body is asserted from the shared
// opcode + the identical result codec rather than from a direct decompile of
// CCashShop::SendCheckDuplicateIDPacket; a single trailing field on the
// cash-shop variant only would not be visible in that evidence.
//
// jms_v185: the name-change feature is absent entirely (derivation.md §1.5),
// so this handler is not registered there even though CHECK_CHAR_NAME exists
// for character creation.
//
// packet-audit:fname CCashShop::SendCheckDuplicateIDPacket
type CheckNameChangeRequest struct {
	name string
}

func NewCheckNameChangeRequest(name string) CheckNameChangeRequest {
	return CheckNameChangeRequest{name: name}
}

// Name is the candidate character name the player typed into the rename
// dialog. Unlike the credential-bearing ops in this package it is not
// sensitive, so String() prints it.
func (m CheckNameChangeRequest) Name() string { return m.name }

func (m CheckNameChangeRequest) Operation() string {
	return CashShopCheckNameChangeHandle
}

func (m CheckNameChangeRequest) String() string {
	return fmt.Sprintf("name [%s]", m.name)
}

func (m CheckNameChangeRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *CheckNameChangeRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
	}
}
