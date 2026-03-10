package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-packet/model"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterMovementWriter = "CharacterMovement"

type CharacterMovementW struct {
	characterId uint32
	movement    model.Movement
}

func NewCharacterMovementW(characterId uint32, movement model.Movement) CharacterMovementW {
	return CharacterMovementW{characterId: characterId, movement: movement}
}

func (m CharacterMovementW) CharacterId() uint32    { return m.characterId }
func (m CharacterMovementW) Movement() model.Movement { return m.movement }
func (m CharacterMovementW) Operation() string       { return CharacterMovementWriter }
func (m CharacterMovementW) String() string          { return fmt.Sprintf("characterId [%d]", m.characterId) }

func (m CharacterMovementW) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *CharacterMovementW) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.movement.Decode(l, ctx)(r, options)
	}
}
