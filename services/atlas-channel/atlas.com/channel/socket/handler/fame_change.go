package handler

import (
	"atlas-channel/fame"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const FameChangeHandle = "FameChangeHandle"

func FameChangeHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		tid := r.ReadUint32()
		mode := r.ReadInt8()
		amount := 2*mode - 1
		l.Debugf("Character [%d] attempting to change [%d] fame by amount [%d]", s.CharacterId(), tid, amount)
		_ = fame.NewProcessor(l, ctx).RequestChange(s.Map(), s.CharacterId(), tid, amount)
	}
}
