package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterDespawnWriter = "CharacterDespawn"

type CharacterDespawn struct {
	characterId uint32
}

func NewCharacterDespawn(characterId uint32) CharacterDespawn {
	return CharacterDespawn{characterId: characterId}
}

func (m CharacterDespawn) CharacterId() uint32 { return m.characterId }
func (m CharacterDespawn) Operation() string   { return CharacterDespawnWriter }
func (m CharacterDespawn) String() string      { return fmt.Sprintf("characterId [%d]", m.characterId) }

func (m CharacterDespawn) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		return w.Bytes()
	}
}

func (m *CharacterDespawn) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
	}
}
