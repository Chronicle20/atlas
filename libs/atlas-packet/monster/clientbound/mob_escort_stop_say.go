package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const MobEscortStopSayWriter = "MobEscortStopSay"

// MobEscortStopSay is the clientbound MOB_ESCORT_STOP_SAY packet
// (CMob::OnEscortStopSay): during an escort stop the server pushes a chat-balloon
// line for the mob to "say". The plan calls this MOB_ESCORT_RETURN_STOP_SAY; the
// registry op name is MOB_ESCORT_STOP_SAY.
//
// Byte layout (IDA-verified):
//   - duration    : int32 — Decode4; stop-act time (m_tEscortStopActTime base)
//   - chatBalloon : int32 — Decode4; chat-balloon style
//   - weather      : bool  — Decode1; suppresses the balloon when set
//   - hasText     : bool  — Decode1; whether a say-string + action follow  } only
//   - text        : string — DecodeStr (length-prefixed ascii)              } present
//   - action      : int32  — Decode4; m_nEscortStopAct                      } when hasText
//
// The text + action sit behind `if (Decode1())`. The codec models the full form
// (hasText=true path) and documents the guard.
//
// IDA basis: CMob::OnEscortStopSay — v95 @0x64c500, jms @0x6f0090
// (Decode4 duration, Decode4 chatBalloon, Decode1 weather, then under Decode1:
// DecodeStr text, Decode4 action). v95/jms only — escort family absent in v83/v84/v87.
//
// packet-audit:fname CMob::OnEscortStopSay
type MobEscortStopSay struct {
	duration    int32
	chatBalloon int32
	weather     bool
	hasText     bool
	text        string
	action      int32
}

func NewMobEscortStopSay(duration int32, chatBalloon int32, weather bool, hasText bool, text string, action int32) MobEscortStopSay {
	return MobEscortStopSay{duration: duration, chatBalloon: chatBalloon, weather: weather, hasText: hasText, text: text, action: action}
}

func (m MobEscortStopSay) Duration() int32    { return m.duration }
func (m MobEscortStopSay) ChatBalloon() int32 { return m.chatBalloon }
func (m MobEscortStopSay) Weather() bool      { return m.weather }
func (m MobEscortStopSay) HasText() bool      { return m.hasText }
func (m MobEscortStopSay) Text() string       { return m.text }
func (m MobEscortStopSay) Action() int32      { return m.action }
func (m MobEscortStopSay) Operation() string  { return MobEscortStopSayWriter }
func (m MobEscortStopSay) String() string {
	return fmt.Sprintf("duration [%d], chatBalloon [%d], weather [%t], hasText [%t], text [%s], action [%d]",
		m.duration, m.chatBalloon, m.weather, m.hasText, m.text, m.action)
}

func (m MobEscortStopSay) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.duration)
		w.WriteInt32(m.chatBalloon)
		w.WriteBool(m.weather)
		w.WriteBool(m.hasText)
		if m.hasText {
			w.WriteAsciiString(m.text)
			w.WriteInt32(m.action)
		}
		return w.Bytes()
	}
}

func (m *MobEscortStopSay) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.duration = r.ReadInt32()
		m.chatBalloon = r.ReadInt32()
		m.weather = r.ReadBool()
		m.hasText = r.ReadBool()
		if m.hasText {
			m.text = r.ReadAsciiString()
			m.action = r.ReadInt32()
		}
	}
}
