package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SheepRanchClothesWriter = "SheepRanchClothes"

type SheepRanchClothes struct {
	characterId uint32
	team        byte
}

func NewSheepRanchClothes(characterId uint32, team byte) SheepRanchClothes {
	return SheepRanchClothes{characterId: characterId, team: team}
}

func (m SheepRanchClothes) CharacterId() uint32 { return m.characterId }
func (m SheepRanchClothes) Team() byte          { return m.team }

func (m SheepRanchClothes) Operation() string { return SheepRanchClothesWriter }
func (m SheepRanchClothes) String() string {
	return fmt.Sprintf("characterId [%d] team [%d]", m.characterId, m.team)
}

func (m SheepRanchClothes) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.team)
		return w.Bytes()
	}
}

func (m *SheepRanchClothes) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.team = r.ReadByte()
	}
}
