package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const SueCharacterHandle = "SueCharacter"

// SueCharacter - CField::SendChatMsgSlash#SueCharacter (opcode varies per version).
// Sent by the /-command parser to report ("sue") a character. The leading field
// is version-branched: v83/v84/v87 lead with the accused character id (int32);
// v95 leads with a sub-command string. Both forms follow with a byte and a
// string. jms is version-absent (no send-site).
// packet-audit:fname CField::SendChatMsgSlash#SueCharacter
type SueCharacter struct {
	characterId uint32 // v83/v84/v87 leading field
	subCommand  string // v95 leading field
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

// SUE_CHARACTER leads with a sub-command string from v95 onward; v83/v84/v87
// lead with the accused character id (int32). The boundary is between 87 and 95
// (v87 uses the legacy form, v95 the new). The guard is written as an inline
// MajorVersion comparison so the packet-audit Atlas analyzer can evaluate it
// per-version (its guard DSL parses t.MajorVersion()-vs-N but not the
// MajorAtLeast/IsRegion helpers). jms is version-absent (no send-site), so its
// branch choice is moot.

func (m SueCharacter) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		if t.MajorVersion() >= 95 {
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
		if t.MajorVersion() >= 95 {
			m.subCommand = r.ReadAsciiString()
		} else {
			m.characterId = r.ReadUint32()
		}
		m.flag = r.ReadByte()
		m.reason = r.ReadAsciiString()
	}
}
