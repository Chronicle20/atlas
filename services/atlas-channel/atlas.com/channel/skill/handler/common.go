package handler

import (
	"atlas-channel/party"
	"context"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func applyToParty(l logrus.FieldLogger) func(ctx context.Context) func(ma _map.Model, characterId uint32, memberBitmap byte) func(idOperator model.Operator[uint32]) error {
	return func(ctx context.Context) func(ma _map.Model, characterId uint32, memberBitmap byte) func(idOperator model.Operator[uint32]) error {
		return func(ma _map.Model, characterId uint32, memberBitmap byte) func(idOperator model.Operator[uint32]) error {
			return func(idOperator model.Operator[uint32]) error {
				if memberBitmap > 0 && memberBitmap < 128 {
					p, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
					if err == nil {
						for _, m := range p.Members() {
							// TODO restrict to those in range, based on bitmap
							if m.Id() != characterId && m.ChannelId() == ma.ChannelId() && m.MapId() == ma.MapId() {
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
