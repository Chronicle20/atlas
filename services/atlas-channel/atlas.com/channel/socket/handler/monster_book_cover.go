package handler

import (
	"atlas-channel/monsterbook"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	mbsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound/monsterbook"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

// MonsterBookCoverHandleFunc decodes a serverbound MonsterBookCover (recv 0x39)
// request and emits a SET_COVER command to atlas-monster-book.
func MonsterBookCoverHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := mbsb.Cover{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		if err := monsterbook.NewProcessor(l, ctx).RequestSetCover(s.CharacterId(), p.CardId()); err != nil {
			l.WithError(err).Errorf("Failed to emit MONSTER_BOOK.SET_COVER for character %d.", s.CharacterId())
		}
	}
}
