package frederick

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// frederickEnvMarkerKey is a test-local context key -- deliberately not
// libs/atlas-env, since frederick sits outside env-domain-guard's permitted
// import list (main.go, kafka/, rest/, socket/) and must not import
// atlas-env even from a test file.
type frederickEnvMarkerKey string

// TestProcessDueNotifications_AppliesEnvContextToNotify pins the review fix:
// this pod's own environment identity must be threaded onto each due
// notification's per-tenant context before the emit. A test with an
// identity envContext would still pass if this were dropped -- decide()
// would then fail open per FR-1.8 and every live deployment, not just this
// pod's, would notify.
func TestProcessDueNotifications_AppliesEnvContextToNotify(t *testing.T) {
	l, _ := test.NewNullLogger()
	n := NotificationEntity{
		Id:           uuid.New(),
		TenantId:     uuid.New(),
		TenantRegion: "GMS",
		TenantMajor:  83,
		TenantMinor:  1,
		CharacterId:  1000,
		NextDay:      2,
	}

	envContext := func(ctx context.Context) context.Context {
		return context.WithValue(ctx, frederickEnvMarkerKey("marker"), "stamped")
	}

	var gotMarker any
	processDueNotifications(l, context.Background(), []NotificationEntity{n}, func(ctx context.Context, _ NotificationEntity) {
		gotMarker = ctx.Value(frederickEnvMarkerKey("marker"))
	}, envContext)

	require.Equal(t, "stamped", gotMarker, "envContext was not applied to the notify context")
}

// TestNotifyAndAdvance_FailedWriteDoesNotEmit pins the DOM-30 fix: the
// tier-notification Kafka message must be buffered into the transactional
// outbox alongside the advance/delete write, so that when the write fails
// the whole transaction rolls back and nothing is enqueued to publish. Before
// this fix the notification was sent via a direct producer call before and
// independently of the write, so a failed write still left the notification
// sent -- the character would be re-notified forever without ever advancing
// a tier.
//
// Uses the same table-drop failure-injection seam as
// shop/processor_test.go's TestRetrieveFrederick_ClearFailure_SkipsOutbox.
func TestNotifyAndAdvance_FailedWriteDoesNotEmit(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, outboxlib.Migration(db))
	l, _ := test.NewNullLogger()

	n := NotificationEntity{
		Id:           uuid.New(),
		TenantId:     uuid.New(),
		TenantRegion: "GMS",
		TenantMajor:  83,
		TenantMinor:  1,
		CharacterId:  1000,
		StoredAt:     time.Now(),
		NextDay:      2,
	}
	require.NoError(t, db.Create(&n).Error)

	// Force the advance/delete write to fail.
	require.NoError(t, db.Migrator().DropTable("frederick_notifications"))

	noTenantCtx := database.WithoutTenantFilter(context.Background())
	notifyAndAdvance(l, db, noTenantCtx)(context.Background(), n)

	var outboxCount int64
	require.NoError(t, db.Model(&outboxlib.Entity{}).Count(&outboxCount).Error)
	assert.Equal(t, int64(0), outboxCount, "no outbox row should be enqueued when the advance/delete write fails")
}

// TestNotifyAndAdvance_SuccessfulWriteEmits is the companion positive case:
// when the write succeeds, the tier notification is enqueued to the outbox
// in the same transaction.
func TestNotifyAndAdvance_SuccessfulWriteEmits(t *testing.T) {
	t.Setenv("EVENT_TOPIC_MERCHANT_STATUS", "EVENT_TOPIC_MERCHANT_STATUS")
	db := setupTestDB(t)
	require.NoError(t, outboxlib.Migration(db))
	l, _ := test.NewNullLogger()

	n := NotificationEntity{
		Id:           uuid.New(),
		TenantId:     uuid.New(),
		TenantRegion: "GMS",
		TenantMajor:  83,
		TenantMinor:  1,
		CharacterId:  1000,
		StoredAt:     time.Now(),
		NextDay:      2,
	}
	require.NoError(t, db.Create(&n).Error)

	noTenantCtx := database.WithoutTenantFilter(context.Background())
	notifyAndAdvance(l, db, noTenantCtx)(context.Background(), n)

	var outboxCount int64
	require.NoError(t, db.Model(&outboxlib.Entity{}).Count(&outboxCount).Error)
	assert.Equal(t, int64(1), outboxCount, "a successful advance should enqueue exactly one outbox row")
}
