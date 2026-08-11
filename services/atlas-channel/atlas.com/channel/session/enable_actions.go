package session

import (
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
)

// EnableActions releases the client's exclusive-request lock with an empty
// StatChanged carrying exclRequestSent=true — the canonical unstick response.
//
// The lock it clears is `CWvsContext::m_bExclRequestSent`, which the v83 client
// arms whenever it sends an exclusive request (item use, skill use, portal
// entry, and — the reason this helper moved out of package handler — a trade
// PUT_ITEM or PUT_MONEY). `CWvsContext::CanSendExclRequest` refuses every
// subsequent request until something clears it, and only three things do: the
// leading exclRequestSent bool of STAT_CHANGED or INVENTORY_OPERATION, or a
// SET_FIELD. There is no client-side timeout.
//
// So an outcome that produces no inventory or stat delta and no field change
// MUST send this, or the client is wedged for the rest of the session. That is
// the class of defect task-205's escrow amendment was written to fix (design
// §5A.6).
//
// The converse is equally load-bearing: do NOT send it for an outcome that
// warps. A successful field change unlocks the client by itself, and unlocking
// it again while it still overlaps the portal rect makes it legitimately
// re-fire the request (see atlas-portals, and task-184).
//
// It lives in package session rather than package handler because both the
// socket handlers and the Kafka consumers need it; two copies had already
// drifted into the tree before this was consolidated.
func EnableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s Model) error {
		return func(wp writer.Producer) func(s Model) error {
			return Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)
		}
	}
}
