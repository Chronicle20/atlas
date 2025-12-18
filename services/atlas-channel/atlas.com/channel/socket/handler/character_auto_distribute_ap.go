package handler

import (
	"atlas-channel/character"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const CharacterAutoDistributeApHandle = "CharacterAutoDistributeApHandle"

func CharacterAutoDistributeApHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		var distributes = make([]character.DistributePacket, 0)

		updateTime := r.ReadUint32()
		nValue := r.ReadUint32()
		for r.Available() >= 8 {
			dwFlag := r.ReadUint32()
			value := r.ReadUint32()
			distributes = append(distributes, character.DistributePacket{
				Flag:  dwFlag,
				Value: value,
			})
		}

		l.Debugf("Reading Auto Distribute AP packet for character [%d]. UpdateTime [%d], nValue [%d].", updateTime, nValue)
		_ = character.NewProcessor(l, ctx).RequestDistributeAp(s.Map(), s.CharacterId(), updateTime, distributes)
	}
}
