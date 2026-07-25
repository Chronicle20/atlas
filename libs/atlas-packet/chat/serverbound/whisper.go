package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const CharacterChatWhisperHandle = "CharacterChatWhisperHandle"

// whisperHasUpdateTime reports whether the leading get_update_time field
// follows the mode byte. The client send-sites CField::SendLocationWhisper /
// CField::SendChatMsgWhisper prove it is present from GMS v87 onward (v83/v84
// omit it) AND in JMS v185 — IDA-verified opcodes/structure:
//
//	gms_v83 (0x78, SendLocationWhisper@0x52f9c6): mode + EncodeStr — NO updateTime
//	gms_v84 (0x7A, SendLocationWhisper@0x53bb1c): mode + EncodeStr — NO updateTime
//	gms_v87 (0x7E, SendChatMsgWhisper@0x556385):  mode + Encode4(get_update_time) + EncodeStr
//	gms_v95 (0x8D, SendLocationWhisper@0x534150): mode + Encode4(get_update_time) + EncodeStr
//	jms_v185 (0x7A, SendLocationWhisper@0x56c73d): mode + Encode4(get_update_time) + EncodeStr
//
// Same v87plus predicate used by interaction/serverbound/operation_chat.go.
func whisperHasUpdateTime(t tenant.Model) bool {
	return (t.Region() == "GMS" && t.MajorVersion() >= 87) || t.Region() == "JMS"
}

type WhisperMode byte

const (
	WhisperModeFind            = WhisperMode(5)
	WhisperModeChat            = WhisperMode(6)
	WhisperModeBuddyWindowFind = WhisperMode(68)
	WhisperModeMacroNotice     = WhisperMode(134)
)

// packet-audit:fname CField::SendChatMsgWhisper
type Whisper struct {
	mode       WhisperMode
	updateTime uint32
	targetName string
	msg        string
}

func (m Whisper) Mode() WhisperMode {
	return m.mode
}

func (m Whisper) UpdateTime() uint32 {
	return m.updateTime
}

func (m Whisper) TargetName() string {
	return m.targetName
}

func (m Whisper) Msg() string {
	return m.msg
}

func (m Whisper) Operation() string {
	return CharacterChatWhisperHandle
}

func (m Whisper) String() string {
	return fmt.Sprintf("mode [%d] updateTime [%d] targetName [%s] msg [%s]", m.mode, m.updateTime, m.targetName, m.msg)
}

func (m Whisper) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	t := tenant.MustFromContext(ctx)
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(m.mode))
		if whisperHasUpdateTime(t) {
			w.WriteInt(m.updateTime)
		}
		w.WriteAsciiString(m.targetName)
		if m.mode == WhisperModeChat {
			w.WriteAsciiString(m.msg)
		}
		return w.Bytes()
	}
}

func (m *Whisper) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = WhisperMode(r.ReadByte())
		if whisperHasUpdateTime(t) {
			m.updateTime = r.ReadUint32()
		}
		m.targetName = r.ReadAsciiString()
		if m.mode == WhisperModeChat {
			m.msg = r.ReadAsciiString()
		}
	}
}
