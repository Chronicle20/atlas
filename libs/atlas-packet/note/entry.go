package note

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// NoteEntry represents a single note in the display list.
type NoteEntry struct {
	Id         uint32
	SenderName string
	Message    string
	Timestamp  time.Time
	Flag       byte
}

func (n NoteEntry) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(n.Id)
		w.WriteAsciiString(n.SenderName + " ")
		w.WriteAsciiString(n.Message)
		w.WriteInt64(model.MsTime(n.Timestamp))
		w.WriteByte(n.Flag)
		return w.Bytes()
	}
}

func (n *NoteEntry) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		n.Id = r.ReadUint32()
		senderWithSpace := r.ReadAsciiString()
		if len(senderWithSpace) > 0 && senderWithSpace[len(senderWithSpace)-1] == ' ' {
			n.SenderName = senderWithSpace[:len(senderWithSpace)-1]
		} else {
			n.SenderName = senderWithSpace
		}
		n.Message = r.ReadAsciiString()
		n.Timestamp = model.FromMsTime(r.ReadInt64())
		n.Flag = r.ReadByte()
	}
}
