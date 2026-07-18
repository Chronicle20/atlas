package opcodes

import (
	"strconv"

	"github.com/sirupsen/logrus"

	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	sw "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
)

// BuildWriterProducer builds a writer.Producer from configuration, filtering to only the writers
// declared in availableWriters. This extracts the common pattern from service main.go files.
func BuildWriterProducer(l logrus.FieldLogger, writers []WriterConfig, availableWriters []string, opWriter socket.OpWriter) sw.Producer {
	rwm := make(map[string]sw.BodyFunc)
	for _, wc := range writers {
		op, err := strconv.ParseUint(wc.OpCode, 0, 16)
		if err != nil {
			l.WithError(err).Errorf("Unable to configure writer [%s] for opcode [%s].", wc.Writer, wc.OpCode)
			continue
		}

		for _, wn := range availableWriters {
			if wn == wc.Writer {
				rwm[wc.Writer] = sw.MessageGetter(opWriter.Write(uint16(op)), wc.Options)
			}
		}
	}
	for _, wn := range availableWriters {
		if _, ok := rwm[wn]; !ok {
			l.Warnf("Service declares writer [%s] but tenant config has no opcode mapping for it.", wn)
		}
	}
	return sw.ProducerGetter(rwm)
}

// HandlerAdapter is a function that adapts a named handler with a validator into a request.Handler.
// Each service provides its own implementation since it depends on the session type.
type HandlerAdapter func(name string, validator interface{}, handler interface{}, options map[string]interface{}) request.Handler

// BuildHandlerMap builds a map of opcode to request.Handler from configuration.
// The validatorMap and handlerMap use interface{} values because the concrete types
// are service-specific (parameterized on session type).
//
// A tenant's socket config carries a single handler list shared by every service
// that speaks the tenant's protocol — login and channel read the same list — so it
// necessarily references handlers this service does not implement (e.g. channel
// item-use/movement handlers appear in the list read by login). Handler entries
// whose name is not in handlerMap therefore belong to another service and are
// skipped silently, rather than logged as a warning per foreign entry on every
// startup.
//
// The inverse case — a handler this service registers but that the config routes no
// opcode to — is NOT surfaced as a warning either: it is common and legitimate.
// Older/partial client versions route only a subset of the service's features, and
// utility handlers (NoOp, Debug) are registered to be wired ad hoc, so warning here
// would just reintroduce version-dependent noise. It is logged at debug for
// diagnosis. Only genuine config errors (missing validator for a routed handler, or
// an unparseable opcode) warn.
func BuildHandlerMap(l logrus.FieldLogger, handlers []HandlerConfig, validatorMap map[string]interface{}, handlerMap map[string]interface{}, adapt HandlerAdapter) map[uint16]request.Handler {
	result := make(map[uint16]request.Handler)
	configured := make(map[string]bool, len(handlerMap))
	for _, hc := range handlers {
		h, ok := handlerMap[hc.Handler]
		if !ok {
			// Handler belongs to another service sharing this tenant socket config.
			continue
		}
		configured[hc.Handler] = true

		v, ok := validatorMap[hc.Validator]
		if !ok {
			l.Warnf("Unable to locate validator [%s] for handler [%s].", hc.Validator, hc.Handler)
			continue
		}

		op, err := strconv.ParseUint(hc.OpCode, 0, 16)
		if err != nil {
			l.WithError(err).Warnf("Unable to configure handler [%s] for opcode [%s].", hc.Handler, hc.OpCode)
			continue
		}

		l.Debugf("Configuring opcode [%s] with validator [%s] and handler [%s].", hc.OpCode, hc.Validator, hc.Handler)
		result[uint16(op)] = adapt(hc.Handler, v, h, hc.Options)
	}

	for name := range handlerMap {
		if !configured[name] {
			l.Debugf("Service registers handler [%s] but tenant config routes no opcode to it.", name)
		}
	}
	return result
}
