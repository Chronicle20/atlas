package outbox

import (
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type notifier struct {
	l   logrus.FieldLogger
	ln  *pq.Listener
	out chan struct{}
}

func newNotifier(l logrus.FieldLogger, dsn string) (*notifier, error) {
	out := make(chan struct{}, 1)
	ln := pq.NewListener(dsn, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			l.WithError(err).Warn("outbox.notify_listener_event")
		}
	})
	if err := ln.Listen(notifyChannel); err != nil {
		_ = ln.Close()
		return nil, err
	}
	n := &notifier{l: l, ln: ln, out: out}
	go n.pump()
	return n, nil
}

func (n *notifier) pump() {
	for ev := range n.ln.Notify {
		_ = ev
		select {
		case n.out <- struct{}{}:
		default:
		}
	}
}

func (n *notifier) C() <-chan struct{} { return n.out }
func (n *notifier) Close()             { _ = n.ln.Close() }
