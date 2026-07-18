package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const CharacterSpouseChatHandle = "CharacterSpouseChatHandle"

// SpouseChat models the SPOUSE_CHAT serverbound packet
// (CUIStatusBar::SendCoupleMessage). The client looks up the partner name from
// the local marriage record, then sends two strings — there is NO leading mode
// byte and NO get_update_time prefix (unlike WHISPER/MULTI_CHAT). IDA-verified
// identical wire across versions (only the opcode shifts):
//
//	gms_v84 (0x7B, @0x9145ee): EncodeStr(spouseName) + EncodeStr(message)
//	gms_v87 (0x7F, @0x953c15): EncodeStr(spouseName) + EncodeStr(message)
//	gms_v95 (0x8E, @0x87b3e0): EncodeStr(spouseName) + EncodeStr(message)
//
// jms_v185 has no SPOUSE_CHAT serverbound op (registry-absent). The clientbound
// counterpart is field/clientbound/SpouseChat (CField::OnCoupleMessage); this
// serverbound struct is named CoupleMessage to avoid the qualified-writer-name
// collision (FieldCoupleMessage vs FieldSpouseChat).
// packet-audit:fname CUIStatusBar::SendCoupleMessage
type CoupleMessage struct {
	spouseName string
	message    string
}

func (m CoupleMessage) SpouseName() string { return m.spouseName }

func (m CoupleMessage) Message() string { return m.message }

func (m CoupleMessage) Operation() string { return CharacterSpouseChatHandle }

func (m CoupleMessage) String() string {
	return fmt.Sprintf("spouseName [%s] message [%s]", m.spouseName, m.message)
}

func (m CoupleMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(m.spouseName) // EncodeStr: partner name from marriage record
		w.WriteAsciiString(m.message)    // EncodeStr: chat message
		return w.Bytes()
	}
}

func (m *CoupleMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.spouseName = r.ReadAsciiString()
		m.message = r.ReadAsciiString()
	}
}
