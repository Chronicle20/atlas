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

type CharacterMovement struct {
	characterId uint32
	movement    model.Movement
}

func NewCharacterMovement(characterId uint32, movement model.Movement) CharacterMovement {
	return CharacterMovement{characterId: characterId, movement: movement}
}

func (m CharacterMovement) CharacterId() uint32    { return m.characterId }
func (m CharacterMovement) Movement() model.Movement { return m.movement }
func (m CharacterMovement) Operation() string       { return CharacterMovementWriter }
func (m CharacterMovement) String() string          { return fmt.Sprintf("characterId [%d]", m.characterId) }

func (m CharacterMovement) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByteArray(m.movement.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func (m *CharacterMovement) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.movement.Decode(l, ctx)(r, options)
	}
}
