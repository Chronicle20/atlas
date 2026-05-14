package monsterbook

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const MonsterBookCoverHandler = "MonsterBookCover"

// Cover - serverbound monster book cover request (recv 0x39). Body is a single int cardId.
type Cover struct {
	cardId uint32
}

func (c *Cover) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		c.cardId = r.ReadUint32()
	}
}

func (c Cover) CardId() uint32    { return c.cardId }
func (c Cover) Operation() string { return MonsterBookCoverHandler }
func (c Cover) String() string    { return fmt.Sprintf("monster book cover cardId [%d]", c.cardId) }
