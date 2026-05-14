package monsterbook

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const MonsterBookSetCoverWriter = "MonsterBookSetCover"

// SetCover - body of monster book cover-changed packet (0x54).
type SetCover struct {
	CardId uint32
}

func (s SetCover) Operation() string { return MonsterBookSetCoverWriter }

func (s SetCover) String() string {
	return fmt.Sprintf("monster book set cover cardId [%d]", s.CardId)
}

func (s SetCover) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(s.CardId)
		return w.Bytes()
	}
}
