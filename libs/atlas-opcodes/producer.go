package opcodes

import (
	"strconv"

	socket "github.com/Chronicle20/atlas-socket"
	"github.com/Chronicle20/atlas-socket/request"
	sw "github.com/Chronicle20/atlas-socket/writer"
	"github.com/sirupsen/logrus"
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
	return sw.ProducerGetter(rwm)
}

// HandlerAdapter is a function that adapts a named handler with a validator into a request.Handler.
// Each service provides its own implementation since it depends on the session type.
type HandlerAdapter func(name string, validator interface{}, handler interface{}, options map[string]interface{}) request.Handler

// BuildHandlerMap builds a map of opcode to request.Handler from configuration.
// The validatorMap and handlerMap use interface{} values because the concrete types
// are service-specific (parameterized on session type).
func BuildHandlerMap(l logrus.FieldLogger, handlers []HandlerConfig, validatorMap map[string]interface{}, handlerMap map[string]interface{}, adapt HandlerAdapter) map[uint16]request.Handler {
	result := make(map[uint16]request.Handler)
	for _, hc := range handlers {
		v, ok := validatorMap[hc.Validator]
		if !ok {
			l.Warnf("Unable to locate validator [%s] for handler [%s].", hc.Validator, hc.Handler)
			continue
		}

		h, ok := handlerMap[hc.Handler]
		if !ok {
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
	return result
}
