package socket

import (
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
)

// shutdownNotice is what a player sees when their channel is drained out
// from under them.
const shutdownNotice = "This channel is shutting down. You will be disconnected shortly; please log back in."

// SessionsForHandle snapshots this channel's sessions for drain phase 2.
// AllInChannelProvider filters the tenant registry by world and channel,
// which is exactly the server.Key triple -- the tenant comes from ctx,
// which is already tenant-scoped by NewListenerContext.
func SessionsForHandle(l logrus.FieldLogger, ctx context.Context, sc server.Model) func() []listener.Session {
	return func() []listener.Session {
		ms, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
		if err != nil {
			l.WithError(err).Warn("listener.drain.sessions_lookup_failed")
			return nil
		}
		out := make([]listener.Session, 0, len(ms))
		for _, m := range ms {
			out = append(out, m)
		}
		return out
	}
}

// KickSession sends the shutdown notice and destroys the session. Destroy
// emits the logout/destroyed Kafka events and then calls
// Model.Disconnect(), which closes the conn -- that is what makes the
// session's run() goroutine return and release the handle's Wg, so drain
// phase 3 completes before its deadline instead of always timing out
// (task-244 design.md §4.5).
func KickSession(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model) func(listener.Session) error {
	sp := session.NewProcessor(l, ctx)
	return func(s listener.Session) error {
		m, ok := s.(session.Model)
		if !ok {
			return fmt.Errorf("unexpected session type %T", s)
		}
		// Best effort -- a write failure must not stop the destroy.
		if err := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(writer.WorldMessagePopUpBody(shutdownNotice))(m); err != nil {
			l.WithError(err).Debug("listener.drain.shutdown_notice_failed")
		}
		return sp.Destroy(m)
	}
}
