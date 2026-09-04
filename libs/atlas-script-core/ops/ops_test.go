package ops

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

func TestDirectResolver(t *testing.T) {
	r := DirectResolver{}

	t.Run("identity", func(t *testing.T) {
		got, err := r.String(1, "p", "PINK_TEXT")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "PINK_TEXT" {
			t.Fatalf("got %q, want %q", got, "PINK_TEXT")
		}
	})

	t.Run("plain int", func(t *testing.T) {
		s, err := r.String(1, "p", "42")
		if err != nil || s != "42" {
			t.Fatalf("String(42) = %q, %v", s, err)
		}
		i, err := r.Int(1, "p", "42")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i != 42 {
			t.Fatalf("got %d, want 42", i)
		}
	})

	t.Run("negative", func(t *testing.T) {
		s, err := r.String(1, "p", "-1")
		if err != nil || s != "-1" {
			t.Fatalf("String(-1) = %q, %v", s, err)
		}
		i, err := r.Int(1, "p", "-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i != -1 {
			t.Fatalf("got %d, want -1", i)
		}
	})

	t.Run("arithmetic", func(t *testing.T) {
		s, err := r.String(1, "p", "10 * 5")
		if err != nil || s != "10 * 5" {
			t.Fatalf("String(10 * 5) = %q, %v", s, err)
		}
		i, err := r.Int(1, "p", "10 * 5")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if i != 50 {
			t.Fatalf("got %d, want 50", i)
		}
	})

	t.Run("not a number", func(t *testing.T) {
		s, err := r.String(1, "p", "abc")
		if err != nil || s != "abc" {
			t.Fatalf("String(abc) = %q, %v", s, err)
		}
		_, err = r.Int(1, "p", "abc")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
	})
}

func TestParamErrorMessage(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		err := missingParam("spawn_monster", "monsterId")
		want := `spawn_monster: parameter "monsterId" is required`
		if err.Error() != want {
			t.Fatalf("got %q, want %q", err.Error(), want)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		err := invalidParam("spawn_monster", "x", "abc", errors.New("value [abc] is not a valid integer"))
		want := `spawn_monster: parameter "x" value "abc": value [abc] is not a valid integer`
		if err.Error() != want {
			t.Fatalf("got %q, want %q", err.Error(), want)
		}
	})

	t.Run("errors.As", func(t *testing.T) {
		var target *ParamError
		err := missingParam("op", "p")
		if !errors.As(err, &target) {
			t.Fatalf("expected errors.As to succeed")
		}
		if target.Param != "p" {
			t.Fatalf("got Param %q, want %q", target.Param, "p")
		}
	})
}

func TestTargetBuilder(t *testing.T) {
	t.Run("field only", func(t *testing.T) {
		f := field.NewBuilder(0, 1, 910010000).Build()
		tgt := NewTargetBuilder(f).Build()
		if tgt.Field().MapId() != 910010000 {
			t.Fatalf("got MapId %d, want 910010000", tgt.Field().MapId())
		}
		x, y, has := tgt.Position()
		if x != 0 || y != 0 || has != false {
			t.Fatalf("got Position %d, %d, %v, want 0, 0, false", x, y, has)
		}
		if tgt.PortalId() != 0 {
			t.Fatalf("got PortalId %d, want 0", tgt.PortalId())
		}
	})

	t.Run("with position", func(t *testing.T) {
		f := field.NewBuilder(0, 1, 910010000).Build()
		tgt := NewTargetBuilder(f).SetPosition(-120, 33).Build()
		x, y, has := tgt.Position()
		if x != -120 || y != 33 || has != true {
			t.Fatalf("got Position %d, %d, %v, want -120, 33, true", x, y, has)
		}
	})

	t.Run("with portal", func(t *testing.T) {
		f := field.NewBuilder(0, 1, 910010000).Build()
		tgt := NewTargetBuilder(f).SetPortalId(7).Build()
		if tgt.PortalId() != 7 {
			t.Fatalf("got PortalId %d, want 7", tgt.PortalId())
		}
	})

	t.Run("with instance", func(t *testing.T) {
		id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		f := field.NewBuilder(0, 1, 910010000).SetInstance(id).Build()
		tgt := NewTargetBuilder(f).Build()
		if tgt.Field().Instance() != id {
			t.Fatalf("got Instance %v, want %v", tgt.Field().Instance(), id)
		}
	})
}

func TestStepAppendTo(t *testing.T) {
	st := newStep(saga.SendMessage, saga.SendMessagePayload{CharacterId: 5})

	b := saga.NewBuilder().SetSagaType(saga.InventoryTransaction).SetInitiatedBy("test")
	s := st.AppendTo(b, "message-5").Build()

	if len(s.Steps) != 1 {
		t.Fatalf("got %d steps, want 1", len(s.Steps))
	}
	if s.Steps[0].StepId != "message-5" {
		t.Fatalf("got StepId %q, want %q", s.Steps[0].StepId, "message-5")
	}
	if s.Steps[0].Status != saga.Pending {
		t.Fatalf("got Status %v, want %v", s.Steps[0].Status, saga.Pending)
	}
	if s.Steps[0].Action != saga.SendMessage {
		t.Fatalf("got Action %v, want %v", s.Steps[0].Action, saga.SendMessage)
	}
}

func TestPayloadOf(t *testing.T) {
	st := newStep(saga.SendMessage, saga.SendMessagePayload{CharacterId: 5})

	p, err := PayloadOf[saga.SendMessagePayload](st)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.CharacterId != 5 {
		t.Fatalf("got CharacterId %d, want 5", p.CharacterId)
	}

	_, err = PayloadOf[saga.SpawnMonsterPayload](st)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestRangeHelpers(t *testing.T) {
	t.Run("rangedInt16", func(t *testing.T) {
		v, err := rangedInt16("op", "p", -120)
		if err != nil || v != int16(-120) {
			t.Fatalf("got %d, %v, want -120, nil", v, err)
		}
		_, err = rangedInt16("op", "p", 40000)
		if err == nil {
			t.Fatalf("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
		if pe.Op != "op" || pe.Param != "p" {
			t.Fatalf("got Op %q Param %q, want op p", pe.Op, pe.Param)
		}
		if !containsSubstring(err.Error(), "out of range for int16") {
			t.Fatalf("got error %q, want substring %q", err.Error(), "out of range for int16")
		}
	})

	t.Run("rangedInt8", func(t *testing.T) {
		v, err := rangedInt8("op", "p", -1)
		if err != nil || v != int8(-1) {
			t.Fatalf("got %d, %v, want -1, nil", v, err)
		}
		_, err = rangedInt8("op", "p", 200)
		if err == nil {
			t.Fatalf("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
		if pe.Op != "op" || pe.Param != "p" {
			t.Fatalf("got Op %q Param %q, want op p", pe.Op, pe.Param)
		}
		if !containsSubstring(err.Error(), "out of range for int8") {
			t.Fatalf("got error %q, want substring %q", err.Error(), "out of range for int8")
		}
	})

	t.Run("rangedUint16", func(t *testing.T) {
		v, err := rangedUint16("op", "p", 65535)
		if err != nil || v != uint16(65535) {
			t.Fatalf("got %d, %v, want 65535, nil", v, err)
		}
		_, err = rangedUint16("op", "p", 65536)
		if err == nil {
			t.Fatalf("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
		if pe.Op != "op" || pe.Param != "p" {
			t.Fatalf("got Op %q Param %q, want op p", pe.Op, pe.Param)
		}
		if !containsSubstring(err.Error(), "out of range for uint16") {
			t.Fatalf("got error %q, want substring %q", err.Error(), "out of range for uint16")
		}
	})

	t.Run("rangedUint32", func(t *testing.T) {
		v, err := rangedUint32("op", "p", 4294967295)
		if err != nil || v != uint32(4294967295) {
			t.Fatalf("got %d, %v, want 4294967295, nil", v, err)
		}
		_, err = rangedUint32("op", "p", -1)
		if err == nil {
			t.Fatalf("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
		if pe.Op != "op" || pe.Param != "p" {
			t.Fatalf("got Op %q Param %q, want op p", pe.Op, pe.Param)
		}
		if !containsSubstring(err.Error(), "out of range for uint32") {
			t.Fatalf("got error %q, want substring %q", err.Error(), "out of range for uint32")
		}
	})

	t.Run("rangedByte", func(t *testing.T) {
		v, err := rangedByte("op", "p", 255)
		if err != nil || v != byte(255) {
			t.Fatalf("got %d, %v, want 255, nil", v, err)
		}
		_, err = rangedByte("op", "p", 256)
		if err == nil {
			t.Fatalf("expected error")
		}
		var pe *ParamError
		if !errors.As(err, &pe) {
			t.Fatalf("expected *ParamError, got %T", err)
		}
		if pe.Op != "op" || pe.Param != "p" {
			t.Fatalf("got Op %q Param %q, want op p", pe.Op, pe.Param)
		}
		if !containsSubstring(err.Error(), "out of range for byte") {
			t.Fatalf("got error %q, want substring %q", err.Error(), "out of range for byte")
		}
	})
}

// recordingResolver records every (param, raw) pair it is asked to resolve,
// then delegates to DirectResolver.
type recordingResolver struct {
	direct  DirectResolver
	params  []string
	rawVals []string
}

func (r *recordingResolver) String(characterId uint32, param string, raw string) (string, error) {
	r.params = append(r.params, param)
	r.rawVals = append(r.rawVals, raw)
	return r.direct.String(characterId, param, raw)
}

func (r *recordingResolver) Int(characterId uint32, param string, raw string) (int, error) {
	r.params = append(r.params, param)
	r.rawVals = append(r.rawVals, raw)
	return r.direct.Int(characterId, param, raw)
}

func TestOptionalIntUsesResolver(t *testing.T) {
	rec := &recordingResolver{}
	got, err := optionalInt(map[string]string{"count": "3"}, rec, 1, "spawn_monster", "count", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 3 {
		t.Fatalf("got %d, want 3", got)
	}
	if len(rec.params) != 1 || rec.params[0] != "count" || rec.rawVals[0] != "3" {
		t.Fatalf("got params %v rawVals %v, want [count] [3]", rec.params, rec.rawVals)
	}
}

func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}
