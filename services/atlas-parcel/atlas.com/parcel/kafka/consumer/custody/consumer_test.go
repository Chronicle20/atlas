package custody

import (
	"atlas-parcel/kafka/message/custody"
	parcelmsg "atlas-parcel/kafka/message/parcel"
	"atlas-parcel/parcel"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	kprod "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// recordedEvent is a decoded event captured by the test producer. It spans
// both envelopes this handler emits: the custody status ack (transactionId +
// type) and the player-facing parcel status event (characterId + type).
type recordedEvent struct {
	transactionId uuid.UUID
	characterId   uint32
	eventType     string
}

// recordingProducer is a test producer.Provider that decodes every emitted
// kafka message into a recordedEvent, so assertions can inspect the ack type
// and transactionId without a live broker. Mirrors
// services/atlas-mts/atlas.com/mts/kafka/consumer/custody/consumer_test.go.
type recordingProducer struct {
	mu     sync.Mutex
	events []recordedEvent
}

func (r *recordingProducer) provider() func(ctx context.Context) kprod.Provider {
	return func(ctx context.Context) kprod.Provider {
		return func(token string) kprod.MessageProducer {
			return func(p model.Provider[[]kafka.Message]) error {
				ms, err := p()
				if err != nil {
					return err
				}
				r.mu.Lock()
				defer r.mu.Unlock()
				for _, m := range ms {
					var ev struct {
						TransactionId uuid.UUID `json:"transactionId"`
						CharacterId   uint32    `json:"characterId"`
						Type          string    `json:"type"`
					}
					if err := json.Unmarshal(m.Value, &ev); err != nil {
						return err
					}
					r.events = append(r.events, recordedEvent{transactionId: ev.TransactionId, characterId: ev.CharacterId, eventType: ev.Type})
				}
				return nil
			}
		}
	}
}

func eventsOfType(events []recordedEvent, eventType string) []recordedEvent {
	var out []recordedEvent
	for _, e := range events {
		if e.eventType == eventType {
			out = append(out, e)
		}
	}
	return out
}

func newTestDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, parcel.Migration)
	db.Logger = logger.Default.LogMode(logger.Silent)
	return db, uuid.New()
}

func newAcceptCommand(transactionId uuid.UUID, parcelId uuid.UUID) custody.Command[custody.AcceptToParcelCommandBody] {
	return custody.Command[custody.AcceptToParcelCommandBody]{
		TransactionId: transactionId,
		Type:          custody.CommandAcceptToParcel,
		Body: custody.AcceptToParcelCommandBody{
			ParcelId:           parcelId,
			CharacterId:        200,
			WorldId:            0,
			SenderAccountId:    1,
			SenderName:         "Sender",
			RecipientId:        100,
			RecipientAccountId: 2,
			RecipientName:      "Bob",
			MesoAmount:         0,
			FeePaid:            5000,
			Quick:              false,
			Message:            "hi",
			ReceivableAt:       time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			ExpiresAt:          time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			HasItem:            true,
			ItemType:           2,
			TemplateId:         1302000,
			Quantity:           1,
			Strength:           5,
			ItemLevel:          3,
			RingId:             777,
			ViciousCount:       12,
			Owner:              "Chronicle",
		},
	}
}

// TestCustodyCommands is table-driven over run: each row exercises a
// distinct custody command sequence (accept/release/restore/remove, replay,
// wrong-recipient) with a distinct assertion set, so the scenario itself is
// carried as a closure rather than shared data fields.
func TestCustodyCommands(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{name: "accept with item", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			transactionId := uuid.New()
			parcelId := uuid.New()
			cmd := newAcceptCommand(transactionId, parcelId)

			handleAcceptToParcel(rp.provider())(db)(l, ctx, cmd)

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			require.NotNil(t, m.ItemId())
			assert.Equal(t, uint32(1302000), *m.ItemId())
			assert.Equal(t, byte(2), m.ItemType(), "ItemType must round-trip from the custody command onto the row")
			assert.Equal(t, uint16(5), m.ItemSnapshot().Strength)
			// task-15 review B1/B2: ItemLevel/RingId/ViciousCount must round-trip
			// through AssetData, not be silently dropped on the parcel snapshot.
			assert.Equal(t, byte(3), m.ItemSnapshot().LevelType, "ItemLevel must map to AssetData.LevelType")
			assert.Equal(t, uint32(777), m.ItemSnapshot().RingId, "RingId must survive the parcel round-trip")
			assert.Equal(t, uint32(12), m.ItemSnapshot().ViciousCount, "ViciousCount must survive the parcel round-trip")
			// RecipientName must round-trip from the custody command onto the row,
			// so an eventual return leg's SenderName (design §7.4) is populated
			// rather than empty.
			assert.Equal(t, "Bob", m.RecipientName())

			accepted := eventsOfType(rp.events, custody.StatusEventAccepted)
			require.Len(t, accepted, 1)
			assert.Equal(t, transactionId, accepted[0].transactionId)

			// accept_to_parcel is parcel_send's last step, so the same
			// delivery must tell the SENDER's channel the send completed —
			// that event is what makes the client's send tab usable again
			// (PARCEL[SUCCESSFULLY_SENT]). Addressed to CharacterId (200,
			// the sender), never to RecipientId (100).
			sent := eventsOfType(rp.events, parcelmsg.StatusEventParcelSent)
			require.Len(t, sent, 1)
			assert.Equal(t, uint32(200), sent[0].characterId)
		}},
		{name: "accept meso only", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			transactionId := uuid.New()
			parcelId := uuid.New()
			cmd := newAcceptCommand(transactionId, parcelId)
			cmd.Body.HasItem = false
			cmd.Body.MesoAmount = 5000

			handleAcceptToParcel(rp.provider())(db)(l, ctx, cmd)

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			assert.Nil(t, m.ItemId())
			assert.Equal(t, uint32(5000), m.MesoAmount())
		}},
		{name: "accept replay", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			transactionId := uuid.New()
			parcelId := uuid.New()
			cmd := newAcceptCommand(transactionId, parcelId)

			handleAcceptToParcel(rp.provider())(db)(l, ctx, cmd)
			handleAcceptToParcel(rp.provider())(db)(l, ctx, cmd)

			all, err := parcel.NewProcessor(l, ctx, db).GetForRecipient(100, 0)
			require.NoError(t, err)
			count := 0
			for _, m := range all {
				if m.Id() == parcelId {
					count++
				}
			}
			assert.Equal(t, 1, count)
		}},
		{name: "release", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			acceptTx := uuid.New()
			parcelId := uuid.New()
			handleAcceptToParcel(rp.provider())(db)(l, ctx, newAcceptCommand(acceptTx, parcelId))

			releaseTx := uuid.New()
			releaseCmd := custody.Command[custody.ReleaseFromParcelCommandBody]{
				TransactionId: releaseTx,
				Type:          custody.CommandReleaseFromParcel,
				Body:          custody.ReleaseFromParcelCommandBody{ParcelId: parcelId, RecipientId: 100},
			}
			handleReleaseFromParcel(rp.provider())(db)(l, ctx, releaseCmd)

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			assert.Equal(t, parcel.StatusReceived, m.Status())
			require.NotNil(t, m.ResolvedAt())

			released := eventsOfType(rp.events, custody.StatusEventReleased)
			require.Len(t, released, 1)
			assert.Equal(t, releaseTx, released[0].transactionId)
		}},
		{name: "release replay", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			parcelId := uuid.New()
			handleAcceptToParcel(rp.provider())(db)(l, ctx, newAcceptCommand(uuid.New(), parcelId))

			releaseTx := uuid.New()
			releaseCmd := custody.Command[custody.ReleaseFromParcelCommandBody]{
				TransactionId: releaseTx,
				Type:          custody.CommandReleaseFromParcel,
				Body:          custody.ReleaseFromParcelCommandBody{ParcelId: parcelId, RecipientId: 100},
			}
			handleReleaseFromParcel(rp.provider())(db)(l, ctx, releaseCmd)
			handleReleaseFromParcel(rp.provider())(db)(l, ctx, releaseCmd)

			released := eventsOfType(rp.events, custody.StatusEventReleased)
			require.Len(t, released, 1, "replay must not re-emit RELEASED")

			errored := eventsOfType(rp.events, custody.StatusEventError)
			assert.Empty(t, errored, "replay must still report success (no ERROR ack)")

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			assert.Equal(t, parcel.StatusReceived, m.Status())
		}},
		{name: "release wrong recipient", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			parcelId := uuid.New()
			handleAcceptToParcel(rp.provider())(db)(l, ctx, newAcceptCommand(uuid.New(), parcelId))

			releaseTx := uuid.New()
			releaseCmd := custody.Command[custody.ReleaseFromParcelCommandBody]{
				TransactionId: releaseTx,
				Type:          custody.CommandReleaseFromParcel,
				Body:          custody.ReleaseFromParcelCommandBody{ParcelId: parcelId, RecipientId: 999},
			}
			handleReleaseFromParcel(rp.provider())(db)(l, ctx, releaseCmd)

			errored := eventsOfType(rp.events, custody.StatusEventError)
			require.Len(t, errored, 1)
			assert.Equal(t, releaseTx, errored[0].transactionId)

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			assert.Equal(t, parcel.StatusPending, m.Status())
		}},
		{name: "restore", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			parcelId := uuid.New()
			handleAcceptToParcel(rp.provider())(db)(l, ctx, newAcceptCommand(uuid.New(), parcelId))
			handleReleaseFromParcel(rp.provider())(db)(l, ctx, custody.Command[custody.ReleaseFromParcelCommandBody]{
				TransactionId: uuid.New(),
				Type:          custody.CommandReleaseFromParcel,
				Body:          custody.ReleaseFromParcelCommandBody{ParcelId: parcelId, RecipientId: 100},
			})

			handleRestoreParcel(rp.provider())(db)(l, ctx, custody.Command[custody.RestoreParcelCommandBody]{
				TransactionId: uuid.New(),
				Type:          custody.CommandRestoreParcel,
				Body:          custody.RestoreParcelCommandBody{ParcelId: parcelId},
			})

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			assert.Equal(t, parcel.StatusPending, m.Status())
			assert.Nil(t, m.ResolvedAt())
		}},
		{name: "restore on a pending row", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			parcelId := uuid.New()
			handleAcceptToParcel(rp.provider())(db)(l, ctx, newAcceptCommand(uuid.New(), parcelId))

			handleRestoreParcel(rp.provider())(db)(l, ctx, custody.Command[custody.RestoreParcelCommandBody]{
				TransactionId: uuid.New(),
				Type:          custody.CommandRestoreParcel,
				Body:          custody.RestoreParcelCommandBody{ParcelId: parcelId},
			})

			errored := eventsOfType(rp.events, custody.StatusEventError)
			assert.Empty(t, errored)

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			assert.Equal(t, parcel.StatusPending, m.Status())
		}},
		{name: "remove", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			parcelId := uuid.New()
			handleAcceptToParcel(rp.provider())(db)(l, ctx, newAcceptCommand(uuid.New(), parcelId))

			handleRemoveParcel(rp.provider())(db)(l, ctx, custody.Command[custody.RemoveParcelCommandBody]{
				TransactionId: uuid.New(),
				Type:          custody.CommandRemoveParcel,
				Body:          custody.RemoveParcelCommandBody{ParcelId: parcelId},
			})

			_, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			assert.ErrorIs(t, err, parcel.ErrNotFound)
		}},
		{name: "remove on a received row", run: func(t *testing.T) {
			db, tid := newTestDB(t)
			ctx := databasetest.TenantContext(tid)
			l, _ := test.NewNullLogger()

			rp := &recordingProducer{}
			parcelId := uuid.New()
			handleAcceptToParcel(rp.provider())(db)(l, ctx, newAcceptCommand(uuid.New(), parcelId))
			handleReleaseFromParcel(rp.provider())(db)(l, ctx, custody.Command[custody.ReleaseFromParcelCommandBody]{
				TransactionId: uuid.New(),
				Type:          custody.CommandReleaseFromParcel,
				Body:          custody.ReleaseFromParcelCommandBody{ParcelId: parcelId, RecipientId: 100},
			})

			handleRemoveParcel(rp.provider())(db)(l, ctx, custody.Command[custody.RemoveParcelCommandBody]{
				TransactionId: uuid.New(),
				Type:          custody.CommandRemoveParcel,
				Body:          custody.RemoveParcelCommandBody{ParcelId: parcelId},
			})

			errored := eventsOfType(rp.events, custody.StatusEventError)
			assert.Empty(t, errored)

			m, err := parcel.NewProcessor(l, ctx, db).GetById(parcelId)
			require.NoError(t, err)
			assert.Equal(t, parcel.StatusReceived, m.Status())
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}
