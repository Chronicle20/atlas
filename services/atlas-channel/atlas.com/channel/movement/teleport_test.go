package movement

import (
	"atlas-channel/character/snapshot"
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

// TestTeleportAndForCharacterPosition covers TeleportCharacter's and
// ForCharacter's last-known-position contract: synchronous registry writes,
// no clientbound broadcast, and the Fh==0 wire pin atlas-character's
// consumer depends on. Each case's setup (announce-spy processor, polling
// deadline, shared Kafka capture) is distinct enough that collapsing them
// into a shared value table would reshape what each asserts, so every row
// carries its scenario as an intact closure.
func TestTeleportAndForCharacterPosition(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			// TeleportCharacter_WritesLastPosition proves TeleportCharacter
			// records the teleported position synchronously so that a
			// plausibility check immediately after the call sees it (no
			// polling needed).
			name: "TeleportCharacter_WritesLastPosition",
			run: func(t *testing.T) {
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
			},
		},
		{
			// TeleportCharacter_NoClientboundBroadcast proves
			// TeleportCharacter never announces to any session -- the
			// client performed the teleport locally, and atlas-character
			// remains the single position authority via the Kafka command
			// (design §4.4).
			name: "TeleportCharacter_NoClientboundBroadcast",
			run: func(t *testing.T) {
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
			},
		},
		{
			// ForCharacter_WritesLastPosition proves ForCharacter records
			// the folded end-of-path position in the same registry
			// TeleportCharacter writes, so an inner-portal plausibility
			// check sees the last known position regardless of how it was
			// set. The write happens inside the async fold/publish
			// goroutine, so this case polls with a bound.
			name: "ForCharacter_WritesLastPosition",
			run: func(t *testing.T) {
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
			},
		},
		{
			// TeleportCharacter_EmitsFhZeroOnWire asserts the wire-level
			// contract of the Kafka command TeleportCharacter publishes: Fh
			// must be 0. atlas-character's consumer branches on fh == 0 to
			// UpdatePosition (preserve the stored foothold) instead of
			// Update (overwrite) -- see b1ddb4db8. An inner-portal teleport
			// carries no foothold from portal data, and inventing one is
			// not an option, so this pins Fh == 0 as a deliberate,
			// load-bearing wire value rather than an oversight the next
			// refactor could silently regress.
			name: "TeleportCharacter_EmitsFhZeroOnWire",
			run: func(t *testing.T) {
				// sharedCapture is installed once for the whole package
				// (testmain_test.go) rather than per-test:
				// producertest.InstallCapturing resets the producer
				// manager singleton, and doing that mid-run would race a
				// still-in-flight async emit from an earlier test's
				// TeleportCharacter/ForCharacter call. Reset here only
				// clears previously recorded messages -- it does not touch
				// the singleton -- so it stays safe to call per-case.
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
			},
		},
		{
			// TeleportCharacter_FeedsSnapshotPositionSynchronously proves
			// TeleportCharacter feeds the character snapshot registry with
			// the teleport destination when an entry already exists, so the
			// attack path never serves a stale pre-teleport position
			// (task-122 post-plan fix).
			name: "TeleportCharacter_FeedsSnapshotPositionSynchronously",
			run: func(t *testing.T) {
				p, tm := newMovementTestProcessor(t)
				f := movementTestField()

				// Entry must exist (feeds never create entries): simulate
				// the lazy populate by viewing it once first.
				_ = snapshot.GetRegistry().View(tm, 9101)

				if err := p.TeleportCharacter(f, 9101, 300, -50); err != nil {
					t.Fatalf("TeleportCharacter returned error: %v", err)
				}

				got := snapshot.GetRegistry().View(tm, 9101)
				if !got.PosValid || got.PosX != 300 || got.PosY != -50 {
					t.Fatalf("position must be fed synchronously by TeleportCharacter: %+v", got)
				}
			},
		},
		{
			// TeleportCharacter_NoEntryNoCreate proves TeleportCharacter
			// never creates a snapshot entry for a character that does not
			// already have one -- the snapshot feed is update-only
			// everywhere on this branch (task-122).
			name: "TeleportCharacter_NoEntryNoCreate",
			run: func(t *testing.T) {
				p, tm := newMovementTestProcessor(t)
				f := movementTestField()

				if err := p.TeleportCharacter(f, 9102, 300, -50); err != nil {
					t.Fatalf("TeleportCharacter returned error: %v", err)
				}

				got := snapshot.GetRegistry().View(tm, 9102)
				if got.PosValid {
					t.Fatalf("TeleportCharacter must never create snapshot entries, got %+v", got)
				}
				if _, ok := snapshot.GetRegistry().ComposedIfValid(tm, 9102); ok {
					t.Fatalf("TeleportCharacter must never create snapshot entries")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
