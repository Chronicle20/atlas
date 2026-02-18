package handler

import (
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

//CMob::Update

const MonsterDamageFriendlyHandle = "MonsterDamageFriendlyHandle"

func MonsterDamageFriendlyHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		attackerId := r.ReadUint32()
		observerId := r.ReadUint32()
		attackedId := r.ReadUint32()
		l.Debugf("Character [%d] observed [%d] attacking [%d].", observerId, attackerId, attackedId)
	}
}
