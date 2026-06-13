package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const SetTamingMobInfoWriter = "SetTamingMobInfo"

type SetTamingMobInfo struct {
	characterId uint32
	level       uint32
	exp         uint32
	tiredness   uint32
	levelUp     bool
}

func NewSetTamingMobInfo(characterId uint32, level uint32, exp uint32, tiredness uint32, levelUp bool) SetTamingMobInfo {
	return SetTamingMobInfo{
		characterId: characterId,
		level:       level,
		exp:         exp,
		tiredness:   tiredness,
		levelUp:     levelUp,
	}
}

func (m SetTamingMobInfo) CharacterId() uint32 { return m.characterId }
func (m SetTamingMobInfo) Level() uint32       { return m.level }
func (m SetTamingMobInfo) Exp() uint32         { return m.exp }
func (m SetTamingMobInfo) Tiredness() uint32   { return m.tiredness }
func (m SetTamingMobInfo) LevelUp() bool       { return m.levelUp }
func (m SetTamingMobInfo) Operation() string   { return SetTamingMobInfoWriter }
func (m SetTamingMobInfo) String() string {
	return fmt.Sprintf("characterId [%d] level [%d] exp [%d] tiredness [%d] levelUp [%t]", m.characterId, m.level, m.exp, m.tiredness, m.levelUp)
}

func (m SetTamingMobInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.level)
		w.WriteInt(m.exp)
		w.WriteInt(m.tiredness)
		w.WriteBool(m.levelUp)
		return w.Bytes()
	}
}
