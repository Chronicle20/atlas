package routine

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/sirupsen/logrus"
)

// Go runs fn in a new goroutine, recovering any panic. A recovered panic is
// logged at Error level with the panic value and full stack trace, then
// swallowed — the goroutine ends and the process continues. ctx is passed
// through to fn unmodified; Go itself never inspects or cancels it.
func Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.WithField("panic", fmt.Sprintf("%v", r)).
					WithField("stack", string(debug.Stack())).
					Errorf("Recovered panic in background goroutine.")
			}
		}()
		fn(ctx)
	}()
}
