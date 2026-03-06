package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type GuildMember struct {
	Name          string
	JobId         uint16
	Level         byte
	Title         byte
	Online        bool
	Signature     uint32
	AllianceTitle byte
}

func (b *GuildMember) Encoder(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		WritePaddedString(w, b.Name, 13)
		w.WriteInt(uint32(b.JobId))
		w.WriteInt(uint32(b.Level))
		w.WriteInt(uint32(b.Title))
		if b.Online {
			w.WriteInt(1)
		} else {
			w.WriteInt(0)
		}
		w.WriteInt(b.Signature)
		w.WriteInt(uint32(b.AllianceTitle))
		return w.Bytes()
	}
}
