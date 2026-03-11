package atlas_packet

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/sirupsen/logrus"
)

// WithResolvedCode resolves a byte code from options at encode time and delegates to the factory-produced encoder.
// This eliminates the need for service-layer wrapper functions that only resolve a code and delegate.
func WithResolvedCode(codeProperty, key string, factory func(byte) packet.Encoder) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
	return func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
		return func(options map[string]interface{}) []byte {
			mode := ResolveCode(l, options, codeProperty, key)
			return factory(mode).Encode(l, ctx)(options)
		}
	}
}

// ResolveCode looks up a byte code from the runtime options map.
// Options are structured as nested maps: options[property][key] = float64(code).
// Returns 99 on any lookup failure (misconfigured opcode — will likely cause a client crash).
func ResolveCode(l logrus.FieldLogger, options map[string]interface{}, property string, key string) byte {
	genericCodes, ok := options[property]
	if !ok {
		l.Errorf("Code [%s] not configured in property [%s]. Defaulting to 99 which will likely cause a client crash.", key, property)
		return 99
	}

	codes, ok := genericCodes.(map[string]interface{})
	if !ok {
		l.Errorf("Code [%s] not configured in property [%s]. Defaulting to 99 which will likely cause a client crash.", key, property)
		return 99
	}

	res, ok := codes[key].(float64)
	if !ok {
		l.Errorf("Code [%s] not configured in property [%s]. Defaulting to 99 which will likely cause a client crash.", key, property)
		return 99
	}
	return byte(res)
}
