package fame

import (
	messageCharacter "atlas-fame/kafka/message/character"
	messageFame "atlas-fame/kafka/message/fame"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	kafkaproducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// capturingWriter records every message written to it, keyed by resolved
// topic name, instead of discarding (producertest.NoopWriter) or hitting a
// real broker. Used to inspect what the DIRECT producer path (rejectEmit
// closures fired outside the outbox-bound mb) actually sends.
type capturingWriter struct {
	topic string
	mu    *sync.Mutex
	msgs  *map[string][]kafka.Message
}

func (w capturingWriter) Topic() string { return w.topic }

func (w capturingWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	(*w.msgs)[w.topic] = append((*w.msgs)[w.topic], msgs...)
	return nil
}

func (w capturingWriter) Close() error { return nil }

// installCapturingProducer swaps the process-wide producer manager singleton
// for one that records messages instead of discarding them, returning the
// captured-messages map and a restore func that must be deferred to put the
// TestMain-installed no-op writer back for subsequent tests.
func installCapturingProducer() (*map[string][]kafka.Message, func()) {
	var mu sync.Mutex
	captured := make(map[string][]kafka.Message)
	kafkaproducer.ResetInstance()
	kafkaproducer.GetManager(kafkaproducer.ConfigWriterFactory(func(topicName string) kafkaproducer.Writer {
		return capturingWriter{topic: topicName, mu: &mu, msgs: &captured}
	}))
	return &captured, func() {
		producertest.InstallNoop()
	}
}

func TestMain(m *testing.M) {
	// RequestChange's rejectEmit closures fire via the DIRECT producer path
	// (D7 fix). Swap in a no-op writer by default so those real-Kafka calls
	// succeed instantly instead of dialing an unreachable broker; individual
	// tests that need to observe the direct path install a capturing writer.
	producertest.InstallNoop()
	os.Exit(m.Run())
}

func setupTestLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l, _ := test.NewNullLogger()
	return l
}

func setupTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}
	return ten
}

func setupTestContext(t *testing.T, ten tenant.Model) context.Context {
	t.Helper()
	return tenant.WithContext(context.Background(), ten)
}

func setupProcessorTestDatabase(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	l := setupTestLogger(t)
	database.RegisterTenantCallbacks(l, db)

	if err = Migration(db); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func TestNewProcessor(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	assert.NotNil(t, p)
}

func TestNewProcessor_ExtractsTenant(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	p := NewProcessor(l, ctx, db)
	impl := p.(*ProcessorImpl)

	assert.Equal(t, ten, impl.t)
}

func TestNewProcessor_PanicsOnMissingTenant(t *testing.T) {
	ctx := context.Background() // No tenant in context
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	assert.Panics(t, func() {
		NewProcessor(l, ctx, db)
	})
}

func TestProcessor_GetByCharacterIdLastMonth_Empty(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestProcessor_GetByCharacterIdLastMonth_ReturnsResults(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	// Create test entity directly in database
	now := time.Now()
	e := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, uint32(1000), result[0].CharacterId())
	assert.Equal(t, uint32(2000), result[0].TargetId())
	assert.Equal(t, int8(1), result[0].Amount())
}

func TestProcessor_GetByCharacterIdLastMonth_FiltersByTenant(t *testing.T) {
	ten1 := setupTestTenant(t)
	ten2 := setupTestTenant(t) // Different tenant
	ctx := setupTestContext(t, ten1)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	now := time.Now()

	// Create entity for tenant 1
	e1 := Entity{
		TenantId:    ten1.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e1)

	// Create entity for tenant 2 (same character ID)
	e2 := Entity{
		TenantId:    ten2.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2001,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, ten1.Id(), result[0].TenantId())
}

func TestProcessor_GetByCharacterIdLastMonth_ExcludesOldRecords(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	now := time.Now()

	// Create recent entity
	e1 := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e1)

	// Create old entity (older than 1 month)
	e2 := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2001,
		Amount:      1,
		CreatedAt:   now.AddDate(0, -2, 0),
	}
	db.Create(&e2)

	p := NewProcessor(l, ctx, db)

	result, err := p.GetByCharacterIdLastMonth(1000)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, uint32(2000), result[0].TargetId())
}

func TestProcessor_ByCharacterIdLastMonthProvider(t *testing.T) {
	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)

	now := time.Now()
	e := Entity{
		TenantId:    ten.Id(),
		Id:          uuid.New(),
		CharacterId: 1000,
		TargetId:    2000,
		Amount:      1,
		CreatedAt:   now.AddDate(0, 0, -5),
	}
	db.Create(&e)

	p := NewProcessor(l, ctx, db)

	provider := p.ByCharacterIdLastMonthProvider(1000)
	result, err := provider()

	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

// TestProcessor_RequestChangeAndEmit_RejectsOnCharacterNotFound exercises the
// failure path fixed by the D7 rejectEmit refactor: a handled validation
// rejection (character lookup fails) commits no fame-log row, so its status
// event must fire on the DIRECT producer path and must NOT be enqueued into
// the outbox alongside a state change that never happened (recipe
// failure-path pitfall #1).
func TestProcessor_RequestChangeAndEmit_RejectsOnCharacterNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")

	captured, restore := installCapturingProducer()
	defer restore()

	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("Failed to migrate outbox table: %v", err)
	}

	p := NewProcessor(l, ctx, db)
	f := field.NewBuilder(world.Id(1), channel.Id(0), _map.Id(100000000)).Build()

	err := p.RequestChangeAndEmit(uuid.New(), f, 1000, 2000, 1)

	// rejectEmit swallows errFameChangeRejected internally; RequestChangeAndEmit
	// reports success once the (handled) rejection has been fired directly.
	assert.NoError(t, err)

	// (a) the rejection fired on the DIRECT path.
	msgs, ok := (*captured)[messageFame.EnvEventTopicFameStatus]
	assert.True(t, ok, "expected a direct-path message for topic %s", messageFame.EnvEventTopicFameStatus)
	assert.Len(t, msgs, 1)

	// (b) no fame log was created (no state change committed).
	result, err := p.GetByCharacterIdLastMonth(1000)
	assert.NoError(t, err)
	assert.Empty(t, result)

	// (c) it did NOT get enqueued into the outbox.
	var count int64
	assert.NoError(t, db.Model(&outbox.Entity{}).Count(&count).Error)
	assert.EqualValues(t, 0, count)
}

// TestProcessor_RequestChangeAndEmit_SuccessEnqueuesOutbox exercises the
// success path: the fame-log write and the resulting REQUEST_CHANGE_FAME
// command to atlas-character are a single committed state change, so per D7
// they must ride the outbox-bound mb (atomic with the write via
// outbox.EmitProvider), not the direct producer path.
func TestProcessor_RequestChangeAndEmit_SuccessEnqueuesOutbox(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
		id := parts[len(parts)-1]
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"` + id + `","attributes":{"name":"Test","level":20}}}`))
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")

	captured, restore := installCapturingProducer()
	defer restore()

	ten := setupTestTenant(t)
	ctx := setupTestContext(t, ten)
	l := setupTestLogger(t)
	db := setupProcessorTestDatabase(t)
	if err := outbox.Migration(db); err != nil {
		t.Fatalf("Failed to migrate outbox table: %v", err)
	}

	p := NewProcessor(l, ctx, db)
	f := field.NewBuilder(world.Id(1), channel.Id(0), _map.Id(100000000)).Build()

	err := p.RequestChangeAndEmit(uuid.New(), f, 1000, 2000, 1)
	assert.NoError(t, err)

	// (a) success does NOT fire on the direct path.
	_, ok := (*captured)[messageFame.EnvEventTopicFameStatus]
	assert.False(t, ok, "success must not fire the status event directly")
	_, ok = (*captured)[messageCharacter.EnvCommandTopic]
	assert.False(t, ok, "success must not fire the character command directly")

	// (b) the fame log WAS created.
	result, err := p.GetByCharacterIdLastMonth(1000)
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	// (c) the REQUEST_CHANGE_FAME command was enqueued into the outbox,
	//     atomic with the fame-log write.
	var entries []outbox.Entity
	assert.NoError(t, db.Find(&entries).Error)
	if assert.Len(t, entries, 1) {
		assert.Equal(t, messageCharacter.EnvCommandTopic, entries[0].Topic)
	}
}
