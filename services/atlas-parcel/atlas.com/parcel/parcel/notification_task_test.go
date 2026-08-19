package parcel

import (
	parcelmsg "atlas-parcel/kafka/message/parcel"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	producertest "github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// notifyClock — T = 2026-03-01T00:00:00Z, the reference instant every
// TestNotificationSweep subtest is written against (brief Step 1).
var notifyClock = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

// notificationCapture is installed once for this package's tests
// (TestMain), mirroring libs/atlas-kafka/producer/producertest's documented
// usage: it swaps the process-wide producer manager away from a real broker
// writer so an emitted PARCEL_ARRIVED can be asserted on without a live
// Kafka.
var notificationCapture *producertest.Capture

func TestMain(m *testing.M) {
	notificationCapture = producertest.InstallCapturing()
	os.Exit(m.Run())
}

// identityEnvContext is a no-op envContext for tests that don't care what
// this pod's environment identity looks like, only that SOME context makes
// it through to the emit.
func identityEnvContext(ctx context.Context) context.Context { return ctx }

func newNotificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	db.Logger = logger.Default.LogMode(logger.Silent)
	return db
}

func newNotificationTestTask(t *testing.T, db *gorm.DB, now time.Time) *NotificationTask {
	t.Helper()
	l, _ := test.NewNullLogger()
	task := NewNotificationTask(l, databasetest.TenantContext(uuid.New()), db, time.Hour, identityEnvContext)
	return task.withClock(func() time.Time { return now })
}

// seedNotifiableParcel inserts a pending-by-default, item-carrying parcel
// scoped to tenantId, receivable in the past and never notified, and returns
// its id.
func seedNotifiableParcel(t *testing.T, db *gorm.DB, tenantId uuid.UUID, mutate func(b *Builder)) uuid.UUID {
	t.Helper()
	id := uuid.New()
	itemId := uint32(1302000)
	b := NewBuilder().
		SetId(id).
		SetWorldId(0).
		SetSenderId(100).
		SetSenderAccountId(1).
		SetSenderName("Alice").
		SetRecipientId(200).
		SetRecipientAccountId(2).
		SetRecipientName("Bob").
		SetStatus(StatusPending).
		SetMesoAmount(0).
		SetFeePaid(800).
		SetItemId(&itemId).
		SetQuantity(1).
		SetCreatedAt(notifyClock.Add(-48 * time.Hour)).
		SetReceivableAt(notifyClock.Add(-time.Hour)).
		SetExpiresAt(notifyClock.Add(30 * 24 * time.Hour))
	if mutate != nil {
		mutate(b)
	}
	m, err := b.Build()
	require.NoError(t, err)

	ctx := databasetest.TenantContext(tenantId)
	_, err = Create(db.WithContext(ctx))(m)
	require.NoError(t, err)
	return id
}

func getNotificationParcel(t *testing.T, db *gorm.DB, tenantId uuid.UUID, id uuid.UUID) Model {
	t.Helper()
	ctx := databasetest.TenantContext(tenantId)
	m, err := ById(id)(db.WithContext(ctx))()
	require.NoError(t, err)
	return m
}

// decodedArrivals decodes every message captured on the parcel status event
// topic into its PARCEL_ARRIVED envelope.
func decodedArrivals(t *testing.T) []parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody] {
	t.Helper()
	msgs := notificationCapture.Messages(parcelmsg.EnvStatusEventTopic)
	var out []parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]
	for _, m := range msgs {
		var ev parcelmsg.StatusEvent[parcelmsg.StatusEventParcelArrivedBody]
		require.NoError(t, json.Unmarshal(m.Value, &ev))
		out = append(out, ev)
	}
	return out
}

func TestNotificationSweep(t *testing.T) {
	t.Run("notifies a newly receivable parcel", func(t *testing.T) {
		notificationCapture.Reset()
		t.Setenv(parcelmsg.EnvStatusEventTopic, parcelmsg.EnvStatusEventTopic)
		db := newNotificationTestDB(t)
		tid := uuid.New()
		id := seedNotifiableParcel(t, db, tid, nil)
		task := newNotificationTestTask(t, db, notifyClock)

		task.Run()

		arrivals := decodedArrivals(t)
		require.Len(t, arrivals, 1)
		assert.Equal(t, parcelmsg.StatusEventParcelArrived, arrivals[0].Type)
		assert.EqualValues(t, 200, arrivals[0].CharacterId)
		assert.Equal(t, "Alice", arrivals[0].Body.SenderName)
		assert.True(t, arrivals[0].Body.HasItem)

		m := getNotificationParcel(t, db, tid, id)
		require.NotNil(t, m.LastNotified())
		assert.True(t, m.LastNotified().Equal(notifyClock))
	})

	t.Run("does not renotify", func(t *testing.T) {
		notificationCapture.Reset()
		t.Setenv(parcelmsg.EnvStatusEventTopic, parcelmsg.EnvStatusEventTopic)
		db := newNotificationTestDB(t)
		tid := uuid.New()
		seedNotifiableParcel(t, db, tid, func(b *Builder) {
			b.SetLastNotified(ptrTime(notifyClock.Add(-time.Hour)))
		})
		task := newNotificationTestTask(t, db, notifyClock)

		task.Run()

		assert.Empty(t, decodedArrivals(t))
	})

	t.Run("not yet receivable", func(t *testing.T) {
		notificationCapture.Reset()
		t.Setenv(parcelmsg.EnvStatusEventTopic, parcelmsg.EnvStatusEventTopic)
		db := newNotificationTestDB(t)
		tid := uuid.New()
		id := seedNotifiableParcel(t, db, tid, func(b *Builder) {
			b.SetReceivableAt(notifyClock.Add(time.Hour))
		})
		task := newNotificationTestTask(t, db, notifyClock)

		task.Run()

		assert.Empty(t, decodedArrivals(t))
		m := getNotificationParcel(t, db, tid, id)
		assert.Nil(t, m.LastNotified())
	})

	t.Run("resolved parcel", func(t *testing.T) {
		notificationCapture.Reset()
		t.Setenv(parcelmsg.EnvStatusEventTopic, parcelmsg.EnvStatusEventTopic)
		db := newNotificationTestDB(t)
		tid := uuid.New()
		seedNotifiableParcel(t, db, tid, func(b *Builder) {
			b.SetStatus(StatusReceived)
		})
		task := newNotificationTestTask(t, db, notifyClock)

		task.Run()

		assert.Empty(t, decodedArrivals(t))
	})

	t.Run("offline recipient still stamps", func(t *testing.T) {
		// No session lookup happens in this sweep at all (FR-24 is served by
		// the OPEN packet's second list, design §7.1), so there is nothing
		// session-shaped to seed here — the point of this subtest is that
		// the sweep does not skip a recipient just because it has no way of
		// knowing whether they are online.
		notificationCapture.Reset()
		t.Setenv(parcelmsg.EnvStatusEventTopic, parcelmsg.EnvStatusEventTopic)
		db := newNotificationTestDB(t)
		tid := uuid.New()
		id := seedNotifiableParcel(t, db, tid, nil)
		task := newNotificationTestTask(t, db, notifyClock)

		task.Run()

		require.Len(t, decodedArrivals(t), 1)
		m := getNotificationParcel(t, db, tid, id)
		require.NotNil(t, m.LastNotified())
	})

	t.Run("topic not configured", func(t *testing.T) {
		notificationCapture.Reset()
		unsetEnv(t, parcelmsg.EnvStatusEventTopic)
		db := newNotificationTestDB(t)
		tid := uuid.New()
		id := seedNotifiableParcel(t, db, tid, nil)
		task := newNotificationTestTask(t, db, notifyClock)

		task.Run()

		assert.Empty(t, decodedArrivals(t))
		m := getNotificationParcel(t, db, tid, id)
		assert.Nil(t, m.LastNotified(), "an unconfigured topic must not stamp a parcel it never notified")
	})

	t.Run("concurrent claim", func(t *testing.T) {
		// This exercises ClaimNotifiable's own compare-and-swap UPDATE
		// directly, sequentially, rather than a genuine simultaneous race:
		// databasetest's sqlite-in-memory harness serializes all access
		// through a single *gorm.DB handle, so it cannot express two
		// replicas' UPDATEs actually overlapping in time — a real race needs
		// two independent DB connections against a shared backing store
		// (production postgres), which this in-memory harness does not
		// have. What IS verified here, faithfully: the SECOND claim attempt
		// against an ALREADY-claimed row affects zero rows — the row-level
		// guard design §8.1 relies on to make concurrent replicas safe
		// without leader election, so at most one PARCEL_ARRIVED is ever
		// emitted for a given parcel.
		db := newNotificationTestDB(t)
		tid := uuid.New()
		seedNotifiableParcel(t, db, tid, nil)

		ctx := databasetest.TenantContext(tid)
		tdb := db.WithContext(ctx)

		first, err := ClaimNotifiable(tdb)(notifyClock, 10)
		require.NoError(t, err)
		require.Len(t, first, 1)

		second, err := ClaimNotifiable(tdb)(notifyClock, 10)
		require.NoError(t, err)
		assert.Len(t, second, 0, "a second claim attempt against an already-claimed row must affect zero rows")
	})
}

func ptrTime(t time.Time) *time.Time { return &t }

// unsetEnv unsets key for the duration of the test, restoring its previous
// value (present or absent) afterward — t.Setenv alone cannot express
// "absent," only "set to this string."
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	prev, ok := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, prev)
		}
	})
}
