package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const TvSetMessageWriter = "TvSetMessage"

// TvSetMessage arms the Maple TV UI: sender look, sender/receiver names, 5
// message lines, the total broadcast wait time, and — when the message is
// addressed to a specific partner — the receiver's AvatarLook (design §1.2,
// IDA v83≡v95).
//
// flag is bit-encoded (bit 2 = receiver look present); Cosmic writes 3 with a
// partner, 1 without. It is computed inside the constructor from whether
// receiverLook is non-nil — callers never pass a raw flag (A1.4: this byte is
// codec-internal, not client-interpreted config, so it is exempt from DOM-25
// resolution).
type TvSetMessage struct {
	flag             byte
	messageType      byte
	senderLook       model.Avatar
	senderName       string
	receiverName     string
	lines            [5]string
	totalWaitSeconds uint32
	receiverLook     *model.Avatar
}

// NewTvSetMessage builds a TvSetMessage. receiverName is always present on
// the wire (empty string when there is no receiver) — it is NOT gated by the
// flag; only the trailing receiverLook bytes are. receiverLook nil means no
// partner: flag resolves to 1 and the trailing look is omitted.
func NewTvSetMessage(messageType byte, senderLook model.Avatar, senderName string, receiverName string, lines [5]string, totalWaitSeconds uint32, receiverLook *model.Avatar) TvSetMessage {
	flag := byte(1)
	if receiverLook != nil {
		flag = 3
	}
	return TvSetMessage{
		flag:             flag,
		messageType:      messageType,
		senderLook:       senderLook,
		senderName:       senderName,
		receiverName:     receiverName,
		lines:            lines,
		totalWaitSeconds: totalWaitSeconds,
		receiverLook:     receiverLook,
	}
}

func (m TvSetMessage) Flag() byte                  { return m.flag }
func (m TvSetMessage) MessageType() byte           { return m.messageType }
func (m TvSetMessage) SenderLook() model.Avatar    { return m.senderLook }
func (m TvSetMessage) SenderName() string          { return m.senderName }
func (m TvSetMessage) ReceiverName() string        { return m.receiverName }
func (m TvSetMessage) Lines() [5]string            { return m.lines }
func (m TvSetMessage) TotalWaitSeconds() uint32    { return m.totalWaitSeconds }
func (m TvSetMessage) ReceiverLook() *model.Avatar { return m.receiverLook }
func (m TvSetMessage) Operation() string           { return TvSetMessageWriter }
func (m TvSetMessage) String() string {
	return fmt.Sprintf("set tv message senderName [%s] receiverName [%s]", m.senderName, m.receiverName)
}

func (m TvSetMessage) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		flag := byte(1)
		if m.receiverLook != nil {
			flag = 3
		}
		w.WriteByte(flag)
		w.WriteByte(m.messageType)
		w.WriteByteArray(m.senderLook.Encode(l, ctx)(options))
		w.WriteAsciiString(m.senderName)
		w.WriteAsciiString(m.receiverName)
		for _, line := range m.lines {
			w.WriteAsciiString(line)
		}
		w.WriteInt(m.totalWaitSeconds)
		if m.receiverLook != nil {
			w.WriteByteArray(m.receiverLook.Encode(l, ctx)(options))
		}
		return w.Bytes()
	}
}

func (m *TvSetMessage) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.flag = r.ReadByte()
		m.messageType = r.ReadByte()
		senderLook := model.Avatar{}
		senderLook.Decode(l, ctx)(r, options)
		m.senderLook = senderLook
		m.senderName = r.ReadAsciiString()
		m.receiverName = r.ReadAsciiString()
		for i := range m.lines {
			m.lines[i] = r.ReadAsciiString()
		}
		m.totalWaitSeconds = r.ReadUint32()
		if m.flag&2 != 0 {
			receiverLook := model.Avatar{}
			receiverLook.Decode(l, ctx)(r, options)
			m.receiverLook = &receiverLook
		} else {
			m.receiverLook = nil
		}
	}
}
