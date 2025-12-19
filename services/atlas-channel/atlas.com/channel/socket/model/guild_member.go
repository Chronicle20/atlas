package model

import (
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/Chronicle20/atlas-tenant"
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

func (b *GuildMember) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
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
	}
}
