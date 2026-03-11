package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const GuildEmblemChangedWriter = "GuildEmblemChanged"

type ForeignEmblemChanged struct {
	characterId         uint32
	logo                uint16
	logoColor           byte
	logoBackground      uint16
	logoBackgroundColor byte
}

func NewForeignEmblemChanged(characterId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) ForeignEmblemChanged {
	return ForeignEmblemChanged{characterId: characterId, logo: logo, logoColor: logoColor, logoBackground: logoBackground, logoBackgroundColor: logoBackgroundColor}
}

func (m ForeignEmblemChanged) Operation() string { return GuildEmblemChangedWriter }
func (m ForeignEmblemChanged) String() string {
	return fmt.Sprintf("characterId [%d], logo [%d], logoColor [%d], logoBackground [%d], logoBackgroundColor [%d]", m.characterId, m.logo, m.logoColor, m.logoBackground, m.logoBackgroundColor)
}

func (m ForeignEmblemChanged) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteShort(m.logoBackground)
		w.WriteByte(m.logoBackgroundColor)
		w.WriteShort(m.logo)
		w.WriteByte(m.logoColor)
		return w.Bytes()
	}
}

func (m *ForeignEmblemChanged) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.logoBackground = r.ReadUint16()
		m.logoBackgroundColor = r.ReadByte()
		m.logo = r.ReadUint16()
		m.logoColor = r.ReadByte()
	}
}
