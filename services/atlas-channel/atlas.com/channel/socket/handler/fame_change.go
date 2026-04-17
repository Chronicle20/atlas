package handler

import (
	"atlas-channel/fame"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	fame2 "github.com/Chronicle20/atlas/libs/atlas-packet/fame/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func FameChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := fame2.Change{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		amount := 2*p.Mode() - 1
		_ = fame.NewProcessor(l, ctx).RequestChange(s.Field(), s.CharacterId(), p.TargetId(), amount)
	}
}
