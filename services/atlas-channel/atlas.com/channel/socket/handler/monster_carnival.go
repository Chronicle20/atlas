package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/monster/carnival/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// MonsterCarnivalHandleFunc handles the serverbound MONSTER_CARNIVAL packet
// (CUIMonsterCarnival::RequestSend): a client carnival action request.
// behavior: deferred (decode-and-log only).
func MonsterCarnivalHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := serverbound.MonsterCarnival{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		// behavior: deferred (decode-and-log only)
	}
}
