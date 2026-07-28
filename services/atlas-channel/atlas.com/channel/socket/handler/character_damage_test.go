package handler

import (
	"atlas-channel/battleship"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"encoding/binary"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	opcodes "github.com/Chronicle20/atlas/libs/atlas-opcodes"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestGaugeCooldownValue(t *testing.T) {
	tests := []struct {
		name      string
		remaining int32
		expected  uint16
	}{
		{"normal", 8500, 8500},
		{"formula max fits (v87+ arm, SLV 10 @ 200)", 29000, 29000},
		{"defensive clamp above uint16", math.MaxUint16 + 1, math.MaxUint16},
		{"defensive floor below zero", -5, 0},
		{"one", 1, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := gaugeCooldownValue(tc.remaining); got != tc.expected {
				t.Errorf("gaugeCooldownValue(%d) = %d, want %d", tc.remaining, got, tc.expected)
			}
		})
	}
}

// TestShouldAnnounceGauge covers the call-site gate in isolation. The full
// handler can't be driven end-to-end in this package's tests (the
// pre-existing, unseamed character.NewProcessor(...).GetById() call returns
// early without a live character service, out of scope for this task), so
// the predicate that stands between a correct and an incorrect announce is
// verified directly against every battleship.DrainStatus value instead.
func TestShouldAnnounceGauge(t *testing.T) {
	tests := []struct {
		name     string
		status   battleship.DrainStatus
		expected bool
	}{
		{"DrainNotRiding does not announce", battleship.DrainNotRiding, false},
		{"DrainSkipped does not announce", battleship.DrainSkipped, false},
		{"DrainDrained announces", battleship.DrainDrained, true},
		{"DrainBroke does not announce", battleship.DrainBroke, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldAnnounceGauge(tc.status); got != tc.expected {
				t.Errorf("shouldAnnounceGauge(%v) = %v, want %v", tc.status, got, tc.expected)
			}
		})
	}
}

// discardConn is a minimal net.Conn stub. announceShipHpGauge's success path
// runs the real session.Announce -> announceEncrypted -> con.Write chain;
// a nil conn (the pattern used by other handler tests that never reach that
// chain) would panic here, so this test needs a live, harmless sink instead.
// The plaintext packet body is captured earlier, from inside
// gaugeProducerRecorder's BodyFunc, so nothing needs to be read back off
// this connection or decrypted.
type discardConn struct{}

func (discardConn) Read(_ []byte) (int, error)         { return 0, io.EOF }
func (discardConn) Write(b []byte) (int, error)        { return len(b), nil }
func (discardConn) Close() error                       { return nil }
func (discardConn) LocalAddr() net.Addr                { return nil }
func (discardConn) RemoteAddr() net.Addr               { return nil }
func (discardConn) SetDeadline(_ time.Time) error      { return nil }
func (discardConn) SetReadDeadline(_ time.Time) error  { return nil }
func (discardConn) SetWriteDeadline(_ time.Time) error { return nil }

var _ net.Conn = discardConn{}

// gaugeProducerRecorder is a fake writer.Producer that records how many
// times, and with what writer name, session.Announce requested a body
// writer. It also captures the plaintext bytes the caller's encoder
// produced, from inside the returned BodyFunc, before session.Announce's own
// encrypt/write step — so a test can assert the exact wire values without
// decrypting anything. A test asserting "no packet announced" checks calls
// == 0: announceShipHpGauge must return before ever invoking the producer.
type gaugeProducerRecorder struct {
	calls    int
	lastName string
	lastBody []byte
}

func (r *gaugeProducerRecorder) producer() writer.Producer {
	return func(name string) (socketwriter.BodyFunc, error) {
		r.calls++
		r.lastName = name
		return func(l logrus.FieldLogger, ctx context.Context) func(encoder packet.Encode) []byte {
			return func(encoder packet.Encode) []byte {
				b := encoder(l, ctx)(nil)
				r.lastBody = b
				return b
			}
		}, nil
	}
}

// newGaugeTestSession builds a v83 GMS tenant + context + session backed by
// discardConn (same mustTenant helper as newCashItemUseTestSession in
// character_cash_item_use_test.go). announceShipHpGauge only reads
// s.CharacterId() for error logging on the (unexercised, since discardConn
// never errors) failure path, so no registry/character-id wiring is needed.
func newGaugeTestSession(t *testing.T) (session.Model, context.Context, uuid.UUID) {
	t.Helper()
	ten := mustTenant(t, "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	s := session.NewSession(uuid.New(), ten, 0, discardConn{})
	return s, ctx, ten.Id()
}

func TestAnnounceShipHpGauge(t *testing.T) {
	const gaugeId = uint32(5221999)

	t.Run("valid options resolve and announce exactly once", func(t *testing.T) {
		s, ctx, tid := newGaugeTestSession(t)
		writer.RegisterTenantWriterOptions(tid, []opcodes.WriterConfig{
			{OpCode: "0xEA", Writer: charpkt.CharacterSkillCooldownWriter, Options: map[string]interface{}{
				"skills": map[string]interface{}{"BATTLESHIP_HP_GAUGE": float64(gaugeId)},
			}},
		})
		t.Cleanup(func() { writer.EvictTenantWriterOptions(tid) })

		rec := &gaugeProducerRecorder{}
		announceShipHpGauge(discardLogger(), ctx, rec.producer(), s, 8500)

		if rec.calls != 1 {
			t.Fatalf("producer calls = %d, want 1", rec.calls)
		}
		if rec.lastName != charpkt.CharacterSkillCooldownWriter {
			t.Errorf("writer name = %q, want %q", rec.lastName, charpkt.CharacterSkillCooldownWriter)
		}
		if len(rec.lastBody) != 6 {
			t.Fatalf("body length = %d, want 6 (uint32 skillId + uint16 cooldown)", len(rec.lastBody))
		}
		if gotSkillId := binary.LittleEndian.Uint32(rec.lastBody[0:4]); gotSkillId != gaugeId {
			t.Errorf("skillId = %d, want %d", gotSkillId, gaugeId)
		}
		if gotCooldown := binary.LittleEndian.Uint16(rec.lastBody[4:6]); gotCooldown != 8500 {
			t.Errorf("cooldown = %d, want %d", gotCooldown, 8500)
		}
	})

	t.Run("no writer options registered for tenant sends nothing", func(t *testing.T) {
		s, ctx, _ := newGaugeTestSession(t)
		// Deliberately do not call writer.RegisterTenantWriterOptions for
		// this tenant at all.

		rec := &gaugeProducerRecorder{}
		announceShipHpGauge(discardLogger(), ctx, rec.producer(), s, 8500)

		if rec.calls != 0 {
			t.Fatalf("producer calls = %d, want 0 (no writer options registered)", rec.calls)
		}
	})

	t.Run("options present but gauge key missing sends nothing", func(t *testing.T) {
		s, ctx, tid := newGaugeTestSession(t)
		writer.RegisterTenantWriterOptions(tid, []opcodes.WriterConfig{
			{OpCode: "0xEA", Writer: charpkt.CharacterSkillCooldownWriter, Options: map[string]interface{}{
				"skills": map[string]interface{}{"SOME_OTHER_SKILL": float64(123)},
			}},
		})
		t.Cleanup(func() { writer.EvictTenantWriterOptions(tid) })

		rec := &gaugeProducerRecorder{}
		announceShipHpGauge(discardLogger(), ctx, rec.producer(), s, 8500)

		if rec.calls != 0 {
			t.Fatalf("producer calls = %d, want 0 (BATTLESHIP_HP_GAUGE key missing from options)", rec.calls)
		}
	})
}
