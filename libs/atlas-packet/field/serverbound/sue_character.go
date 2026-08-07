package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const SueCharacterHandle = "SueCharacter"

// SueCharacter - CField::SendChatMsgSlash#SueCharacter (opcode varies per version).
// Sent by the /-command parser to report ("sue") a character. The leading field
// is version-branched: v83/v84/v87 lead with the accused character id (int32);
// v92 onward leads with a sub-command string. Both forms follow with a byte and
// a string. jms is version-absent (no send-site) — see
// docs/tasks/task-145-player-reports/packet-findings.md §7.4 for the full
// cross-version derivation and the JMS absence evidence.
// packet-audit:fname CField::SendChatMsgSlash#SueCharacter
type SueCharacter struct {
	characterId uint32 // v83/v84/v87 leading field
	subCommand  string // v92+ leading field
	flag        byte
	reason      string
}

func NewSueCharacterLegacy(characterId uint32, flag byte, reason string) SueCharacter {
	return SueCharacter{characterId: characterId, flag: flag, reason: reason}
}

func NewSueCharacterV95(subCommand string, flag byte, reason string) SueCharacter {
	return SueCharacter{subCommand: subCommand, flag: flag, reason: reason}
}

func (m SueCharacter) CharacterId() uint32 { return m.characterId }
func (m SueCharacter) SubCommand() string  { return m.subCommand }
func (m SueCharacter) Flag() byte          { return m.flag }
func (m SueCharacter) Reason() string      { return m.reason }

func (m SueCharacter) Operation() string {
	return SueCharacterHandle
}

func (m SueCharacter) String() string {
	return fmt.Sprintf("characterId [%d], subCommand [%s], flag [%d], reason [%s]", m.characterId, m.subCommand, m.flag, m.reason)
}

// SUE_CHARACTER leads with a sub-command string from v92 onward; v83/v84/v87
// lead with the accused character id (int32). The boundary is v87|v92, not
// v87|v95 — re-derived directly from IDA across all three verified send-sites
// (task-30, docs/tasks/task-145-player-reports/packet-findings.md §7.4):
//
//	v87 CField::SendChatMsgSlash#SueCharacter @0x553526 (GMSv87_4GB.exe): `push 75h` (opcode 117) → Encode4([edi+1444h]) → Encode1(esi) → EncodeStr — legacy int32-charId lead.
//	v92 same function @0x53b7d0 (GMS_v92_1_DEVM.exe): `push 7Dh` (opcode 125) → EncodeStr(target) → Encode1(0-5 range) → EncodeStr(reason) — string lead, at the identical stack offset (ebp+68h+0x9c) as v95's confirmed sSubCmd.
//	v95 same function @0x5413e5 (GMS_v95.0_U_DEVM.exe): `push 7Eh` (opcode 126) → EncodeStr(sSubCmd) → Encode1(edi) → EncodeStr(reason) — string lead, confirming the form v92 already uses.
//
// v92 was previously undocumented (the boundary was only pinned to "between 87
// and 95"); this was a live wire-format bug, not just a stale comment — v92 is
// wired into seed templates by a later task on this branch (task-18), and the
// old `>= 95` gate would have sent v92 traffic down the legacy int32 path,
// decoding a leading string as a raw uint32.
//
// The guard is written as an inline MajorVersion comparison (not the
// MajorAtLeast helper) so the packet-audit Atlas analyzer can evaluate it
// per-version: its guard DSL (tools/packet-audit/internal/atlaspacket/guard.go
// compileBinary) only recognizes `t.MajorVersion()`/`t.MinorVersion()`/
// `t.Region()` wrapped in a binary comparison — a bare call like
// `t.MajorAtLeast(92)` isn't a *ast.BinaryExpr and falls through
// compileExpr's unsupported-expression case, so guardFromIf silently
// substitutes an always-true guard instead of failing loudly. That would make
// the analyzer treat this branch as applying to every version, not just
// v92+, silently corrupting whatever consumes the guard (coverage matrix /
// gate-check). `>= N` is also the CI-whitelisted idiom form (docs/packets/
// PROCESS.md's gate-lint only flags strict `> N` and inclusive `<= N`, not
// `>= N`/`< N`), so this is not the "raw comparison the guards flag" pattern.
// jms is version-absent (no send-site), so its branch choice is moot.

func (m SueCharacter) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if t.MajorVersion() >= 92 {
			w.WriteAsciiString(m.subCommand)
		} else {
			w.WriteInt(m.characterId)
		}
		w.WriteByte(m.flag)
		w.WriteAsciiString(m.reason)
		return w.Bytes()
	}
}

func (m *SueCharacter) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		if t.MajorVersion() >= 92 {
			m.subCommand = r.ReadAsciiString()
		} else {
			m.characterId = r.ReadUint32()
		}
		m.flag = r.ReadByte()
		m.reason = r.ReadAsciiString()
	}
}
