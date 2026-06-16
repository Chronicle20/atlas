package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MobSpeakingWriter = "MobSpeaking"

// MobSpeaking is the clientbound MOB_SPEAKING packet (CMob::OnMobSpeaking): the
// server tells the client a mob should "speak" (trigger a speech/animation pair);
// the two ints are forwarded verbatim to CMob::TrySpeaking(J, J).
//
// Byte layout (IDA-verified, two Decode4):
//   - speechType : int32 — first Decode4 (TrySpeaking arg 1)
//   - action     : int32 — second Decode4 (TrySpeaking arg 2)
//
// IDA basis: CMob::OnMobSpeaking — v83 @0x6711ea, v84 @0x687743, v87 @0x6ac31e,
// v95 @0x650000, jms @0x6ee398 (`v3 = Decode4; v4 = Decode4; TrySpeaking(v3, v4)`).
type MobSpeaking struct {
	speechType int32
	action     int32
}

func NewMobSpeaking(speechType int32, action int32) MobSpeaking {
	return MobSpeaking{speechType: speechType, action: action}
}

func (m MobSpeaking) SpeechType() int32  { return m.speechType }
func (m MobSpeaking) Action() int32      { return m.action }
func (m MobSpeaking) Operation() string  { return MobSpeakingWriter }
func (m MobSpeaking) String() string {
	return fmt.Sprintf("speechType [%d], action [%d]", m.speechType, m.action)
}

func (m MobSpeaking) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt32(m.speechType)
		w.WriteInt32(m.action)
		return w.Bytes()
	}
}

func (m *MobSpeaking) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.speechType = r.ReadInt32()
		m.action = r.ReadInt32()
	}
}
