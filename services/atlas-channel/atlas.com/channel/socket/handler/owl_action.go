package handler

import (
	"atlas-channel/session"
	"atlas-channel/shopscanner"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	atlas_packet "github.com/Chronicle20/atlas/libs/atlas-packet"
	merchantsb "github.com/Chronicle20/atlas/libs/atlas-packet/merchant/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// OwlActionHandleFunc answers CUIShopScanner::OnCreate (mode OPEN, the only
// mode the client ever sends — task-127 design §1.3) with the most-searched
// hot list. The expected mode byte is config-resolved from the handler
// entry's options.operations table, never hard-coded.
func OwlActionHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := merchantsb.OwlAction{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		expected := atlas_packet.ResolveCode(l, readerOptions, "operations", "OPEN")
		if p.Mode() != expected {
			l.Warnf("Character [%d] sent owl action with unexpected mode [%d], expected [%d].", s.CharacterId(), p.Mode(), expected)
			return
		}
		if !_map.IsFreeMarketRoom(s.MapId()) {
			l.Warnf("Character [%d] sent owl action outside the Free Market (map [%d]).", s.CharacterId(), s.MapId())
			return
		}
		_ = shopscanner.NewProcessor(l, ctx).SendHotList(wp)(s)
	}
}
