package atlas_packet

import (
	"context"
	"math"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
)

// CodeConfigured reports whether options[property][key] is present, without
// resolving it and without logging. It is the quiet companion to ResolveCode,
// for the case where an ABSENT key is a legitimate, expected state rather than
// a misconfiguration: a dispatcher arm that a given client version simply does
// not have (the tenant template is the per-version authority — DOM-25, so
// "not in the operations table" IS "not on this version", and no version
// number is hard-coded anywhere).
//
// Callers must skip the write when this returns false. Sending anyway routes
// through ResolveCode's 99 sentinel, which the client has no arm for.
// ResolveCode/ResolveValue stay loud because for them a miss is a real
// misconfiguration; this predicate exists so a by-design absence does not have
// to be logged as an error on every occurrence.
func CodeConfigured(options map[string]interface{}, property string, key string) bool {
	genericCodes, ok := options[property]
	if !ok {
		return false
	}
	codes, ok := genericCodes.(map[string]interface{})
	if !ok {
		return false
	}
	_, ok = codes[key]
	return ok
}

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

// ResolveCode16 looks up an optional uint16 code from the runtime options map.
// Unlike ResolveCode — which returns a loud 99 default because a missing mode
// byte is a fatal misconfiguration — a miss here is soft: the caller decides
// what absence means. Used for sparse bit tables (e.g. the petSkill usPetSkill
// table) where an unverified bit must encode as absent, never a guessed value.
func ResolveCode16(l logrus.FieldLogger, options map[string]interface{}, property string, key string) (uint16, bool) {
	genericCodes, ok := options[property]
	if !ok {
		l.Debugf("Property [%s] missing from options when resolving code [%s].", property, key)
		return 0, false
	}

	codes, ok := genericCodes.(map[string]interface{})
	if !ok {
		l.Debugf("Property [%s] is not a map when resolving code [%s].", property, key)
		return 0, false
	}

	raw, ok := codes[key]
	if !ok {
		l.Debugf("Code [%s] not configured in property [%s].", key, property)
		return 0, false
	}

	switch v := raw.(type) {
	case float64:
		if v < 0 || v > math.MaxUint16 {
			l.Debugf("Code [%s] in property [%s] has out-of-range value %.0f (valid range 0-%d).", key, property, v, math.MaxUint16)
			return 0, false
		}
		return uint16(v), true
	case string:
		n, err := strconv.ParseUint(v, 0, 16)
		if err != nil {
			l.WithError(err).Debugf("Code [%s] in property [%s] has unparseable value [%q].", key, property, v)
			return 0, false
		}
		return uint16(n), true
	default:
		l.Debugf("Code [%s] in property [%s] has unsupported type %T.", key, property, raw)
		return 0, false
	}
}

// ResolveValue looks up a uint32 wire value from the runtime options map.
// Same nested-map format as ResolveCode: options[property][key] = value,
// where the value may be a JSON number (float64) or a string parsable by
// strconv.ParseUint with base 0 (e.g. "0x4FAE6F"). Unlike ResolveCode there
// is no safe sentinel for a 4-byte wire value, so any miss logs an error and
// returns (0, false); callers must skip the write rather than send a guess.
func ResolveValue(l logrus.FieldLogger, options map[string]interface{}, property string, key string) (uint32, bool) {
	genericValues, ok := options[property]
	if !ok {
		l.Errorf("Property [%s] missing from options when resolving value [%s].", property, key)
		return 0, false
	}

	values, ok := genericValues.(map[string]interface{})
	if !ok {
		l.Errorf("Property [%s] is not a map when resolving value [%s].", property, key)
		return 0, false
	}

	raw, ok := values[key]
	if !ok {
		l.Errorf("Value [%s] not configured in property [%s].", key, property)
		return 0, false
	}

	switch v := raw.(type) {
	case float64:
		return uint32(v), true
	case string:
		n, err := strconv.ParseUint(v, 0, 32)
		if err != nil {
			l.WithError(err).Errorf("Value [%s] in property [%s] has unparseable value [%q].", key, property, v)
			return 0, false
		}
		return uint32(n), true
	default:
		l.Errorf("Value [%s] in property [%s] has unsupported type %T.", key, property, raw)
		return 0, false
	}
}
