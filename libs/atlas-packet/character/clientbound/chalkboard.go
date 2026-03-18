package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const ChalkboardUseWriter = "ChalkboardUse"

type ChalkboardUse struct {
	characterId uint32
	active      bool
	message     string
}

func NewChalkboardUse(characterId uint32, message string) ChalkboardUse {
	return ChalkboardUse{characterId: characterId, active: true, message: message}
}

func NewChalkboardClear(characterId uint32) ChalkboardUse {
	return ChalkboardUse{characterId: characterId, active: false}
}

func (m ChalkboardUse) CharacterId() uint32 { return m.characterId }
func (m ChalkboardUse) Active() bool        { return m.active }
func (m ChalkboardUse) Message() string     { return m.message }
func (m ChalkboardUse) Operation() string   { return ChalkboardUseWriter }
func (m ChalkboardUse) String() string {
	return fmt.Sprintf("characterId [%d], active [%t]", m.characterId, m.active)
}

func (m ChalkboardUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteBool(m.active)
		if m.active {
			w.WriteAsciiString(m.message)
		}
		return w.Bytes()
	}
}

func (m *ChalkboardUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.active = r.ReadBool()
		if m.active {
			m.message = r.ReadAsciiString()
		}
	}
}
