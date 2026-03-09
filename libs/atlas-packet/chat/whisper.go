package chat

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterChatWhisperHandle = "CharacterChatWhisperHandle"

type WhisperMode byte

const (
	WhisperModeFind            = WhisperMode(5)
	WhisperModeChat            = WhisperMode(6)
	WhisperModeBuddyWindowFind = WhisperMode(68)
	WhisperModeMacroNotice     = WhisperMode(134)
)

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
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
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
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			m.updateTime = r.ReadUint32()
		}
		m.targetName = r.ReadAsciiString()
		if m.mode == WhisperModeChat {
			m.msg = r.ReadAsciiString()
		}
	}
}
