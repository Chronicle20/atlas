package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterEffectForeignWriter = "CharacterEffectForeign"

// EffectForeign - a generic wrapper that prepends a characterId to any inner effect's encoded bytes.
type EffectForeign struct {
	characterId uint32
	innerBytes  []byte
}

func NewEffectForeign(characterId uint32, innerBytes []byte) EffectForeign {
	return EffectForeign{characterId: characterId, innerBytes: innerBytes}
}

func (m EffectForeign) CharacterId() uint32 { return m.characterId }
func (m EffectForeign) InnerBytes() []byte  { return m.innerBytes }
func (m EffectForeign) Operation() string   { return CharacterEffectForeignWriter }

func (m EffectForeign) String() string {
	return fmt.Sprintf("foreign effect characterId [%d]", m.characterId)
}

func (m EffectForeign) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByteArray(m.innerBytes)
		return w.Bytes()
	}
}

func (m *EffectForeign) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: server-send-only
	}
}
