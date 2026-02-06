package model

import (
	"time"

	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
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

func (n *Note) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(n.Id)
		w.WriteAsciiString(n.SenderName + " ")
		w.WriteAsciiString(n.Message)
		w.WriteInt64(msTime(n.Timestamp))
		w.WriteByte(n.Flag)
	}
}

func msTime(t time.Time) int64 {
	if t.IsZero() {
		return -1
	}
	return t.Unix()*int64(10000000) + int64(116444736000000000)
}
