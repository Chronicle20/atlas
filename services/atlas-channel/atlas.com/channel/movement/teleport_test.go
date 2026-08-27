package movement

import (
	movement2 "atlas-channel/kafka/message/movement"
	"atlas-channel/position"
	"atlas-channel/socket/writer"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestTeleportCharacter_WritesLastPosition proves TeleportCharacter records
// the teleported position synchronously so that a plausibility check
// immediately after the call sees it (no polling needed).
func TestTeleportCharacter_WritesLastPosition(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()

	if err := p.TeleportCharacter(f, 42, 300, -50); err != nil {
		t.Fatalf("TeleportCharacter returned error: %v", err)
	}

	got, ok := position.GetRegistry().Lookup(tm, 42)
	if !ok {
		t.Fatalf("expected registry entry for character 42, found none")
	}
	if got.X != 300 || got.Y != -50 {
		t.Fatalf("got position %+v, want {300 -50}", got)
	}
}

// TestTeleportCharacter_NoClientboundBroadcast proves TeleportCharacter never
// announces to any session — the client performed the teleport locally, and
// atlas-character remains the single position authority via the Kafka
// command (design §4.4).
func TestTeleportCharacter_NoClientboundBroadcast(t *testing.T) {
	tm := newMovementTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	announces := 0
	spy := writer.Producer(func(name string) (writer.BodyFunc, error) {
		announces++
		return nil, nil
	})

	p := NewProcessor(logrus.New(), ctx, spy).(*ProcessorImpl)
	f := movementTestField()

	if err := p.TeleportCharacter(f, 42, 300, -50); err != nil {
		t.Fatalf("TeleportCharacter returned error: %v", err)
	}

	if announces != 0 {
		t.Fatalf("TeleportCharacter recorded %d announces, want 0", announces)
	}
}

// TestForCharacter_WritesLastPosition proves ForCharacter records the folded
// end-of-path position in the same registry TeleportCharacter writes, so an
// inner-portal plausibility check sees the last known position regardless of
// how it was set. The write happens inside the async fold/publish
// goroutine, so this test polls with a bound.
func TestForCharacter_WritesLastPosition(t *testing.T) {
	p, tm := newMovementTestProcessor(t)
	f := movementTestField()

	mv := packetmodel.Movement{
		StartX: 0,
		StartY: 0,
		Elements: []packetmodel.MovementCodec{
			&packetmodel.NormalElement{Element: packetmodel.Element{X: 150, Y: 250, Fh: 1, BMoveAction: 0}},
		},
	}

	const characterId = 4242

	if err := p.ForCharacter(f, characterId, mv); err != nil {
		t.Fatalf("ForCharacter returned error: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for {
		got, ok := position.GetRegistry().Lookup(tm, characterId)
		if ok {
			if got.X != 150 || got.Y != 250 {
				t.Fatalf("got position %+v, want {150 250}", got)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for ForCharacter to write the position registry")
		}
		time.Sleep(time.Millisecond)
	}
}

// TestTeleportCharacter_EmitsFhZeroOnWire asserts the wire-level contract of
// the Kafka command TeleportCharacter publishes: Fh must be 0. atlas-character's
// consumer branches on fh == 0 to UpdatePosition (preserve the stored foothold)
// instead of Update (overwrite) -- see b1ddb4db8. An inner-portal teleport
// carries no foothold from portal data, and inventing one is not an option, so
// this pins Fh == 0 as a deliberate, load-bearing wire value rather than an
// oversight the next refactor could silently regress.
func TestTeleportCharacter_EmitsFhZeroOnWire(t *testing.T) {
	// sharedCapture is installed once for the whole package (testmain_test.go)
	// rather than per-test: producertest.InstallCapturing resets the producer
	// manager singleton, and doing that mid-run would race a still-in-flight
	// async emit from an earlier test's TeleportCharacter/ForCharacter call.
	// Reset here only clears previously recorded messages -- it does not touch
	// the singleton -- so it stays safe to call per-test.
	sharedCapture.Reset()

	p, _ := newMovementTestProcessor(t)
	f := movementTestField()

	if err := p.TeleportCharacter(f, 42, 300, -50); err != nil {
		t.Fatalf("TeleportCharacter returned error: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for {
		got := sharedCapture.Messages(movement2.EnvCommandCharacterMovement)
		if len(got) > 0 {
			var cmd movement2.Command[any]
			if err := json.Unmarshal(got[0].Value, &cmd); err != nil {
				t.Fatalf("unmarshal COMMAND_TOPIC_CHARACTER_MOVEMENT command: %v", err)
			}
			if cmd.Fh != 0 {
				t.Fatalf("got Fh=%d, want 0 (foothold must be preserved, not overwritten, on an inner-portal teleport)", cmd.Fh)
			}
			if cmd.X != 300 || cmd.Y != -50 {
				t.Fatalf("got (X,Y)=(%d,%d), want (300,-50)", cmd.X, cmd.Y)
			}
			if cmd.ObjectId != 42 {
				t.Fatalf("got ObjectId=%d, want 42", cmd.ObjectId)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for TeleportCharacter to emit the movement command")
		}
		time.Sleep(time.Millisecond)
	}
}
