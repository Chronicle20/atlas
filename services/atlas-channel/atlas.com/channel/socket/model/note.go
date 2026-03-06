package model

import (
	"context"
	"time"

	"github.com/Chronicle20/atlas-socket/response"
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
		w.WriteInt64(msTime(n.Timestamp))
		w.WriteByte(n.Flag)
		return w.Bytes()
	}
}

func msTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}
