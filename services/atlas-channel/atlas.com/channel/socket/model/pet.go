package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/response"
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

func (b *Pet) Encoder(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(b.TemplateId)
		w.WriteAsciiString(b.Name)
		w.WriteLong(uint64(b.Id))
		w.WriteInt16(b.X)
		w.WriteInt16(b.Y)
		w.WriteByte(b.Stance)
		w.WriteInt16(b.Foothold)
		w.WriteByte(b.NameTag)
		w.WriteByte(b.ChatBalloon)
		return w.Bytes()
	}
}
