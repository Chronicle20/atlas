package handler

import (
	"atlas-channel/character/teleportrock"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	trsb "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// teleportRockRequestsFunc allows tests to capture the emitted commands.
var teleportRockRequestsFunc = func(l logrus.FieldLogger, ctx context.Context) teleportrock.Processor {
	return teleportrock.NewProcessor(l, ctx)
}

// TeleportRockAddMapHandleFunc handles TROCK_ADD_MAP
// (CWvsContext::SendMapTransferRequest). Register carries no map id — the
// current map comes from server-side session state (design §1 Q1). All client
// feedback rides the status-event consumer (fire-and-forget here).
func TeleportRockAddMapHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := trsb.AddMap{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		proc := teleportRockRequestsFunc(l, ctx)
		if p.Register() {
			if err := proc.RequestAddMap(s.Field(), s.CharacterId(), p.Vip()); err != nil {
				l.WithError(err).Errorf("Unable to request map registration for character [%d].", s.CharacterId())
			}
			return
		}
		if err := proc.RequestRemoveMap(s.Field().WorldId(), s.CharacterId(), _map.Id(p.MapId()), p.Vip()); err != nil {
			l.WithError(err).Errorf("Unable to request map removal for character [%d].", s.CharacterId())
		}
	}
}
