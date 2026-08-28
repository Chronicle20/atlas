package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MapleLifeCheckNameHandle = "MapleLifeCheckNameHandle"

// CheckName is the serverbound duplicate-name probe for the Maple Life
// (Cash/0543 character-creation, CUICharacterSaleDlg) naming step —
// CUICharacterSaleDlg::SendCheckDuplicateIDPacket. Its answer is
// maplelife/clientbound.MapleLifeResult (CUICharacterSaleDlg::OnCheckDuplicatedIDResult).
//
// Unlike the cash-shop rename probe (cash/serverbound.CheckNameChangeRequest,
// which shares CHECK_CHAR_NAME's opcode with the login-socket character
// creation duplicate check), the Maple Life probe has its OWN
// Maple-Life-specific opcode on every in-scope version — no collision with
// CHECK_CHAR_NAME(21) was found on any of them (derivation.md §6.1-§6.2,
// routing consequence (A)):
//
//	gms_v83  opcode 256 (0x100)
//	gms_v84  opcode 263 (0x107) — CUICharacterSaleDlg exists on this binary
//	         and was mis-flagged VERSION-ABSENT by an earlier, retracted pass
//	         (derivation.md §2.0-CORRECTION, supersedes §2.0/§6.1)
//	gms_v87  opcode 270 (0x10E)
//	gms_v92  opcode 301 (0x12D)
//	gms_v95  opcode 311
//
// Body, identical on every in-scope version:
//
//	EncodeStr  sCharName
//
// The client validates the candidate name locally via
// is_valid_character_name(sCharName, !isUnderCover) BEFORE sending —
// invalid names never reach the wire (derivation.md §6.1).
//
// Derivation: docs/tasks/task-246-maple-life-character-creation/derivation.md §6.
// packet-audit:fname CUICharacterSaleDlg::SendCheckDuplicateIDPacket
type CheckName struct {
	name string
}

func NewCheckName(name string) CheckName {
	return CheckName{name: name}
}

// Name is the candidate character name the player typed into the Maple
// Life naming dialog.
func (m CheckName) Name() string { return m.name }

func (m CheckName) Operation() string {
	return MapleLifeCheckNameHandle
}

func (m CheckName) String() string {
	return fmt.Sprintf("name [%s]", m.name)
}

func (m CheckName) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *CheckName) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.name = r.ReadAsciiString()
	}
}
