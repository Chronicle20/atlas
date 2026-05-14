package monsterbook

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterBookSetCardWriter = "MonsterBookSetCard"

// SetCard - body of monster book card-add packet (0x53).
type SetCard struct {
	CardId uint32
	Level  uint8
	Added  bool
}

func (s SetCard) Operation() string { return MonsterBookSetCardWriter }

func (s SetCard) String() string {
	return fmt.Sprintf("monster book set card cardId [%d] level [%d] added [%v]", s.CardId, s.Level, s.Added)
}

func (s SetCard) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		var flag byte
		if s.Added {
			flag = 1
		}
		w.WriteByte(flag)
		w.WriteInt(s.CardId)
		w.WriteInt(uint32(s.Level))
		return w.Bytes()
	}
}
