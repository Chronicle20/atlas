package guild

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

type SetEmblem struct {
	logoBackground      uint16
	logoBackgroundColor byte
	logo                uint16
	logoColor           byte
}

func (m SetEmblem) LogoBackground() uint16      { return m.logoBackground }
func (m SetEmblem) LogoBackgroundColor() byte    { return m.logoBackgroundColor }
func (m SetEmblem) Logo() uint16                 { return m.logo }
func (m SetEmblem) LogoColor() byte              { return m.logoColor }

func (m SetEmblem) Operation() string { return "SetEmblem" }

func (m SetEmblem) String() string {
	return fmt.Sprintf("logoBackground [%d] logoBackgroundColor [%d] logo [%d] logoColor [%d]", m.logoBackground, m.logoBackgroundColor, m.logo, m.logoColor)
}

func (m SetEmblem) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteShort(m.logoBackground)
		w.WriteByte(m.logoBackgroundColor)
		w.WriteShort(m.logo)
		w.WriteByte(m.logoColor)
		return w.Bytes()
	}
}

func (m *SetEmblem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.logoBackground = r.ReadUint16()
		m.logoBackgroundColor = r.ReadByte()
		m.logo = r.ReadUint16()
		m.logoColor = r.ReadByte()
	}
}
