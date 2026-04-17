package model

import (
	"context"
	"time"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Note represents a note for a character
type Note struct {
	Id         uint32
	SenderName string
	Message    string
	Timestamp  time.Time
	Flag       byte
}

func (n *Note) Encoder(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(n.Id)
		w.WriteAsciiString(n.SenderName + " ")
		w.WriteAsciiString(n.Message)
		w.WriteInt64(packetmodel.MsTime(n.Timestamp))
		w.WriteByte(n.Flag)
		return w.Bytes()
	}
}
