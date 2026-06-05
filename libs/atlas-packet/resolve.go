package atlas_packet

import (
	"context"
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
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
// Options are structured as nested maps: options[property][key] = code. The code may be
// a JSON number (decoded as float64) or a string parsable by strconv.ParseUint with base 0
// (e.g. "0x01"), matching the format used by WriterConfig.OpCode.
// Returns 99 on any lookup failure (misconfigured opcode — will likely cause a client crash).
func ResolveCode(l logrus.FieldLogger, options map[string]interface{}, property string, key string) byte {
	genericCodes, ok := options[property]
	if !ok {
		l.Errorf("Property [%s] missing from options when resolving code [%s]. Defaulting to 99 which will likely cause a client crash.", property, key)
		return 99
	}

	codes, ok := genericCodes.(map[string]interface{})
	if !ok {
		l.Errorf("Property [%s] is not a map when resolving code [%s]. Defaulting to 99 which will likely cause a client crash.", property, key)
		return 99
	}

	raw, ok := codes[key]
	if !ok {
		l.Errorf("Code [%s] not configured in property [%s]. Defaulting to 99 which will likely cause a client crash.", key, property)
		return 99
	}

	switch v := raw.(type) {
	case float64:
		return byte(v)
	case string:
		n, err := strconv.ParseUint(v, 0, 8)
		if err != nil {
			l.WithError(err).Errorf("Code [%s] in property [%s] has unparseable value [%q]. Defaulting to 99 which will likely cause a client crash.", key, property, v)
			return 99
		}
		return byte(n)
	default:
		l.Errorf("Code [%s] in property [%s] has unsupported type %T. Defaulting to 99 which will likely cause a client crash.", key, property, raw)
		return 99
	}
}

// ResolveName is the inverse of ResolveCode: given a wire byte, it returns the
// configured key whose value equals that byte. Inbound handlers receive a byte
// the client echoed back (e.g. lastMessageType) and must map it to a semantic
// name before classifying it. Values are matched using the same float64/string
// (base-0) encodings ResolveCode accepts. Returns ("", false) on any miss so
// callers can apply a safe default rather than crash the client.
func ResolveName(l logrus.FieldLogger, options map[string]interface{}, property string, code byte) (string, bool) {
	genericCodes, ok := options[property]
	if !ok {
		l.Debugf("Property [%s] missing from options when reverse-resolving code [%d].", property, code)
		return "", false
	}

	codes, ok := genericCodes.(map[string]interface{})
	if !ok {
		l.Debugf("Property [%s] is not a map when reverse-resolving code [%d].", property, code)
		return "", false
	}

	for name, raw := range codes {
		switch v := raw.(type) {
		case float64:
			if byte(v) == code {
				return name, true
			}
		case string:
			if n, err := strconv.ParseUint(v, 0, 8); err == nil && byte(n) == code {
				return name, true
			}
		}
	}
	return "", false
}
