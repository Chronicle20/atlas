// Package ops holds the shared building blocks (Step, Target, Resolver,
// ParamError, param/range helpers) that every script operation is built
// from. Operations built on this package perform no network, Redis, Kafka
// or REST I/O, never call a saga processor directly, and never log; they
// return errors and let the caller decide what to do with them.
package ops

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	scriptcontext "github.com/Chronicle20/atlas/libs/atlas-script-core/context"
)

// now is the package clock. Tests override it to make expiration-bearing
// payloads deterministic.
var now = time.Now

// Step is a single saga step produced by an operation: a status (always
// saga.Pending until AppendTo hands it to a saga.Builder), the saga.Action
// it performs, and its payload.
type Step struct {
	status  saga.Status
	action  saga.Action
	payload any
}

// newStep builds a Step in the Pending state.
func newStep(action saga.Action, payload any) Step {
	return Step{status: saga.Pending, action: action, payload: payload}
}

func (s Step) Status() saga.Status { return s.status }
func (s Step) Action() saga.Action { return s.action }
func (s Step) Payload() any        { return s.payload }

// AppendTo adds the step to a saga builder under the caller's step id.
// Step-id composition stays with the caller (FR-8).
func (s Step) AppendTo(b *saga.Builder, id string) *saga.Builder {
	return b.AddStep(id, s.status, s.action, s.payload)
}

// PayloadOf type-asserts a step's payload. Callers whose step-id format embeds
// a parsed field (map-actions' "spawn-%d-%d" uses the monster id) use this
// rather than re-parsing the param.
func PayloadOf[T any](s Step) (T, error) {
	p, ok := s.payload.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("step payload is %T, not %T", s.payload, zero)
	}
	return p, nil
}

// Target is a resolved location an operation acts on: a field, an optional
// (x, y) position, and an optional portal id.
type Target struct {
	field       field.Model
	x           int16
	y           int16
	hasPosition bool
	portalId    uint32
}

// TargetBuilder builds a Target.
type TargetBuilder struct {
	field       field.Model
	x           int16
	y           int16
	hasPosition bool
	portalId    uint32
}

// NewTargetBuilder seeds a TargetBuilder with the target field.
func NewTargetBuilder(f field.Model) *TargetBuilder {
	return &TargetBuilder{field: f}
}

// SetPosition sets the target's (x, y) coordinates.
func (b *TargetBuilder) SetPosition(x, y int16) *TargetBuilder {
	b.x = x
	b.y = y
	b.hasPosition = true
	return b
}

// SetPortalId sets the target's portal id.
func (b *TargetBuilder) SetPortalId(id uint32) *TargetBuilder {
	b.portalId = id
	return b
}

// Build returns the built Target.
func (b *TargetBuilder) Build() Target {
	return Target{
		field:       b.field,
		x:           b.x,
		y:           b.y,
		hasPosition: b.hasPosition,
		portalId:    b.portalId,
	}
}

func (t Target) Field() field.Model { return t.field }

// Position returns the target's coordinates and whether they were set.
func (t Target) Position() (int16, int16, bool) { return t.x, t.y, t.hasPosition }

func (t Target) PortalId() uint32 { return t.portalId }

// Resolver resolves a raw operation parameter value, given the acting
// character and the parameter name. Implementations may consult
// conversation state (e.g. NPC dialogue context) or resolve directly.
type Resolver interface {
	String(characterId uint32, param string, raw string) (string, error)
	Int(characterId uint32, param string, raw string) (int, error)
}

// DirectResolver resolves without conversation state: String is the identity,
// Int is context.EvaluateValueAsInt (which supports arithmetic expressions).
// Used by map-actions, reactor-actions and portal-actions.
type DirectResolver struct{}

func (DirectResolver) String(_ uint32, _ string, raw string) (string, error) { return raw, nil }
func (DirectResolver) Int(_ uint32, _ string, raw string) (int, error) {
	return scriptcontext.EvaluateValueAsInt(raw)
}

// ParamError is a hard failure resolving or validating a single operation
// parameter. Err is nil for a missing required parameter, non-nil when a
// present value failed to parse or fell outside its valid range.
type ParamError struct {
	Op    string
	Param string
	Value string
	Err   error
}

func (e *ParamError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: parameter %q is required", e.Op, e.Param)
	}
	return fmt.Sprintf("%s: parameter %q value %q: %v", e.Op, e.Param, e.Value, e.Err)
}

func (e *ParamError) Unwrap() error { return e.Err }

func missingParam(op, name string) error { return &ParamError{Op: op, Param: name} }

func invalidParam(op, name, value string, err error) error {
	return &ParamError{Op: op, Param: name, Value: value, Err: err}
}

// requiredString resolves a required string parameter, failing with
// missingParam when absent or invalidParam when the resolver errors.
func requiredString(p map[string]string, r Resolver, cid uint32, op, name string) (string, error) {
	raw, ok := p[name]
	if !ok {
		return "", missingParam(op, name)
	}
	v, err := r.String(cid, name, raw)
	if err != nil {
		return "", invalidParam(op, name, raw, err)
	}
	return v, nil
}

// optionalString resolves an optional string parameter, returning def when
// absent.
func optionalString(p map[string]string, r Resolver, cid uint32, op, name, def string) (string, error) {
	raw, ok := p[name]
	if !ok {
		return def, nil
	}
	v, err := r.String(cid, name, raw)
	if err != nil {
		return "", invalidParam(op, name, raw, err)
	}
	return v, nil
}

// requiredInt resolves a required int parameter, failing with missingParam
// when absent or invalidParam when the resolver errors.
func requiredInt(p map[string]string, r Resolver, cid uint32, op, name string) (int, error) {
	raw, ok := p[name]
	if !ok {
		return 0, missingParam(op, name)
	}
	v, err := r.Int(cid, name, raw)
	if err != nil {
		return 0, invalidParam(op, name, raw, err)
	}
	return v, nil
}

// optionalInt resolves an optional int parameter, returning def when absent.
func optionalInt(p map[string]string, r Resolver, cid uint32, op, name string, def int) (int, error) {
	raw, ok := p[name]
	if !ok {
		return def, nil
	}
	v, err := r.Int(cid, name, raw)
	if err != nil {
		return 0, invalidParam(op, name, raw, err)
	}
	return v, nil
}

// rangedInt8 narrows v to int8, failing with invalidParam when v is out of range.
func rangedInt8(op, name string, v int) (int8, error) {
	if v < math.MinInt8 || v > math.MaxInt8 {
		return 0, invalidParam(op, name, strconv.Itoa(v), fmt.Errorf("out of range for int8"))
	}
	return int8(v), nil
}

// rangedInt16 narrows v to int16, failing with invalidParam when v is out of range.
func rangedInt16(op, name string, v int) (int16, error) {
	if v < math.MinInt16 || v > math.MaxInt16 {
		return 0, invalidParam(op, name, strconv.Itoa(v), fmt.Errorf("out of range for int16"))
	}
	return int16(v), nil
}

// rangedByte narrows v to byte, failing with invalidParam when v is out of range.
func rangedByte(op, name string, v int) (byte, error) {
	if v < 0 || v > math.MaxUint8 {
		return 0, invalidParam(op, name, strconv.Itoa(v), fmt.Errorf("out of range for byte"))
	}
	return byte(v), nil
}

// rangedUint16 narrows v to uint16, failing with invalidParam when v is out of range.
func rangedUint16(op, name string, v int) (uint16, error) {
	if v < 0 || v > math.MaxUint16 {
		return 0, invalidParam(op, name, strconv.Itoa(v), fmt.Errorf("out of range for uint16"))
	}
	return uint16(v), nil
}

// rangedUint32 narrows v to uint32, failing with invalidParam when v is out of range.
func rangedUint32(op, name string, v int) (uint32, error) {
	if v < 0 || v > math.MaxUint32 {
		return 0, invalidParam(op, name, strconv.Itoa(v), fmt.Errorf("out of range for uint32"))
	}
	return uint32(v), nil
}
