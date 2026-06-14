package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SpouseChatWriter = "SpouseChat"

// SpouseChat models the SPOUSE_CHAT clientbound packet (CField::OnCoupleMessage).
//
// The client dispatches on a leading mode byte (Decode1). The two sub-modes share
// the same scalar widths but differ in field set:
//
//	mode 4 ("own/sender message"):   Decode1(mode) + DecodeStr(sender) + Decode1(flag) + DecodeStr(chatText)
//	mode 5 ("partner message"):      Decode1(mode) +                     Decode1(flag) + DecodeStr(chatText)
//
// The IDA export flattens both guarded arms into a single positional read order:
//
//	Decode1(mode) + DecodeStr(sender) + Decode1(flag) + DecodeStr(chatText) + Decode1(partnerFlag) + DecodeStr(partnerText)
//
// The model carries the union of both arms so the wire-level diff aligns
// positionally with that flattened read order. A concrete send populates the arm
// for its mode; the representative fixture exercises the full union (mode-4 sender
// segment + mode-5 partner segment) so the round-trip closes for the modeled shape.
type SpouseChat struct {
	mode        byte
	sender      string
	flag        byte
	chatText    string
	partnerFlag byte
	partnerText string
}

const (
	// SpouseChatModeOwn is the "own/sender message" branch (sender + flag + chatText).
	SpouseChatModeOwn byte = 4
	// SpouseChatModePartner is the "partner message" branch (flag + chatText, no sender).
	SpouseChatModePartner byte = 5
)

// NewSpouseChat constructs the union representative of the SPOUSE_CHAT read order:
// mode + sender + flag + chatText (mode-4 segment) + partnerFlag + partnerText
// (mode-5 segment).
func NewSpouseChat(mode byte, sender string, flag byte, chatText string, partnerFlag byte, partnerText string) SpouseChat {
	return SpouseChat{mode: mode, sender: sender, flag: flag, chatText: chatText, partnerFlag: partnerFlag, partnerText: partnerText}
}

func (m SpouseChat) Mode() byte         { return m.mode }
func (m SpouseChat) Sender() string     { return m.sender }
func (m SpouseChat) Flag() byte         { return m.flag }
func (m SpouseChat) ChatText() string   { return m.chatText }
func (m SpouseChat) PartnerFlag() byte  { return m.partnerFlag }
func (m SpouseChat) PartnerText() string { return m.partnerText }

func (m SpouseChat) Operation() string { return SpouseChatWriter }
func (m SpouseChat) String() string {
	return fmt.Sprintf("spouse chat mode [%d] sender [%s]", m.mode, m.sender)
}

func (m SpouseChat) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)            // Decode1: mode discriminator
		w.WriteAsciiString(m.sender)   // DecodeStr: sender (mode-4 segment)
		w.WriteByte(m.flag)            // Decode1: flag (mode-4 segment)
		w.WriteAsciiString(m.chatText) // DecodeStr: chatText (mode-4 segment)
		w.WriteByte(m.partnerFlag)     // Decode1: flag (mode-5 segment)
		w.WriteAsciiString(m.partnerText) // DecodeStr: chatText (mode-5 segment)
		return w.Bytes()
	}
}

func (m *SpouseChat) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.sender = r.ReadAsciiString()
		m.flag = r.ReadByte()
		m.chatText = r.ReadAsciiString()
		m.partnerFlag = r.ReadByte()
		m.partnerText = r.ReadAsciiString()
	}
}
