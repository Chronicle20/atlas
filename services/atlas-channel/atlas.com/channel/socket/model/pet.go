package model

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Pet struct {
	TemplateId  uint32
	Name        string
	Id          uint32
	X           int16
	Y           int16
	Stance      byte
	Foothold    int16
	NameTag     byte
	ChatBalloon byte
}

func (b *Pet) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(b.TemplateId)
		w.WriteAsciiString(b.Name)
		w.WriteLong(uint64(b.Id))
		w.WriteInt16(b.X)
		w.WriteInt16(b.Y)
		w.WriteByte(b.Stance)
		w.WriteInt16(b.Foothold)
		w.WriteByte(b.NameTag)
		w.WriteByte(b.ChatBalloon)
	}
}
