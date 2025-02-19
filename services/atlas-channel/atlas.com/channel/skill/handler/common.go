package handler

import (
	"atlas-channel/party"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func applyToParty(l logrus.FieldLogger) func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, memberBitmap byte) func(idOperator model.Operator[uint32]) error {
	return func(ctx context.Context) func(worldId byte, channelId byte, mapId uint32, characterId uint32, memberBitmap byte) func(idOperator model.Operator[uint32]) error {
		return func(worldId byte, channelId byte, mapId uint32, characterId uint32, memberBitmap byte) func(idOperator model.Operator[uint32]) error {
			return func(idOperator model.Operator[uint32]) error {
				if memberBitmap > 0 && memberBitmap < 128 {
					p, err := party.GetByMemberId(l)(ctx)(characterId)
					if err == nil {
						for _, m := range p.Members() {
							// TODO restrict to those in range, based on bitmap
							if m.Id() != characterId && m.ChannelId() == channelId && m.MapId() == mapId {
								_ = idOperator(m.Id())
							}
						}
					}
				}
				return nil
			}
		}
	}
}
