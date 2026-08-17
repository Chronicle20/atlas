package services

import (
	"atlas-configurations/scope"
	"atlas-configurations/services/service"
	"atlas-configurations/services/task"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// testEntity is a SQLite-compatible version of Entity for testing
type testEntity struct {
	Id          uuid.UUID       `gorm:"type:text;primaryKey"`
	Type        ServiceType     `gorm:"type:varchar"`
	Data        json.RawMessage `gorm:"type:text;not null"`
	Environment string          `gorm:"not null;default:''"`
}

func (testEntity) TableName() string {
	return "services"
}

// testHistoryEntity is a SQLite-compatible version of HistoryEntity for
// testing. update()/delete() always write a history row, so any test that
// exercises those paths needs this table migrated.
type testHistoryEntity struct {
	Id          uuid.UUID       `gorm:"type:text;primaryKey"`
	ServiceId   uuid.UUID       `gorm:"type:text"`
	Type        ServiceType     `gorm:"type:varchar"`
	Data        json.RawMessage `gorm:"type:text;not null"`
	CreatedAt   time.Time       `gorm:"not null"`
	Environment string          `gorm:"not null;default:''"`
}

func (testHistoryEntity) TableName() string {
	return "service_history"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// :memory: is per-connection; the default pool can open more than one
	// connection and silently query an empty database.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	// Use SQLite-compatible schema
	err = db.AutoMigrate(&testEntity{}, &testHistoryEntity{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
}

// testDatabase is an alias for setupTestDB matching the brief's naming
// (task-232 R13-3).
func testDatabase(t *testing.T) *gorm.DB {
	return setupTestDB(t)
}

// envContext installs a registry (task-232 R13-2) knowing "main" and
// "pr-123" - both baselined to "main" - and returns a context carrying
// caller as the operation's environment. The registry is process-wide, so
// the previous one is restored on cleanup.
func envContext(t *testing.T, caller string) context.Context {
	t.Helper()
	reg := env.NewMapRegistry(env.Id(caller), nil)
	for _, e := range []string{"main", "pr-123"} {
		reg.Apply(env.Record{Name: env.Id(e), Baseline: env.Id("main"), Phase: env.PhaseActive})
	}
	prev := env.CurrentRegistry()
	env.SetRegistry(reg)
	t.Cleanup(func() { env.SetRegistry(prev) })
	return env.WithContext(context.Background(), env.Id(caller))
}

// seedService inserts a service row directly at the Entity level, owned by
// environment, and returns it.
func seedService(t *testing.T, db *gorm.DB, environment string, serviceType ServiceType) testEntity {
	t.Helper()
	e := testEntity{
		Id:          uuid.New(),
		Type:        serviceType,
		Data:        json.RawMessage(`{"type":"login-service","tasks":[]}`),
		Environment: environment,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatalf("failed to seed service: %v", err)
	}
	return e
}

// patchWithNewTenant builds an update body distinct from seedService's
// seeded data, so a successful (wrongly-authorized) write would be
// detectable as a byte change.
func patchWithNewTenant(t *testing.T) service.InputRestModel {
	t.Helper()
	return service.InputRestModel{
		Type:    string(ServiceTypeLogin),
		Tenants: json.RawMessage(`[{"id":"new-tenant","port":9999}]`),
	}
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func createLoginEntity(db *gorm.DB, t *testing.T) uuid.UUID {
	id := uuid.New()
	loginData := service.LoginRestModel{
		Tasks: []task.RestModel{
			{Type: "heartbeat", Interval: 10000, Duration: 0},
		},
		Tenants: []service.LoginTenantRestModel{
			{Id: "tenant-1", Port: 8484},
		},
	}
	jsonData, err := json.Marshal(loginData)
	if err != nil {
		t.Fatalf("failed to marshal login data: %v", err)
	}

	entity := &Entity{
		Id:   id,
		Type: ServiceTypeLogin,
		Data: jsonData,
	}
	err = db.Create(entity).Error
	if err != nil {
		t.Fatalf("failed to create login entity: %v", err)
	}
	return id
}

func createChannelEntity(db *gorm.DB, t *testing.T) uuid.UUID {
	id := uuid.New()
	channelData := service.ChannelRestModel{
		Tasks: []task.RestModel{
			{Type: "respawn", Interval: 5000, Duration: 0},
		},
		Tenants: []service.ChannelTenantRestModel{
			{
				Id:        "tenant-1",
				IPAddress: "127.0.0.1",
				Worlds: []service.ChannelWorldRestModel{
					{
						Id: 0,
						Channels: []service.ChannelChannelRestModel{
							{Id: 0, Port: 7575},
						},
					},
				},
			},
		},
	}
	jsonData, err := json.Marshal(channelData)
	if err != nil {
		t.Fatalf("failed to marshal channel data: %v", err)
	}

	entity := &Entity{
		Id:   id,
		Type: ServiceTypeChannel,
		Data: jsonData,
	}
	err = db.Create(entity).Error
	if err != nil {
		t.Fatalf("failed to create channel entity: %v", err)
	}
	return id
}

func createDropsEntity(db *gorm.DB, t *testing.T) uuid.UUID {
	id := uuid.New()
	dropsData := service.GenericRestModel{
		Tasks: []task.RestModel{
			{Type: "cleanup", Interval: 60000, Duration: 0},
		},
	}
	jsonData, err := json.Marshal(dropsData)
	if err != nil {
		t.Fatalf("failed to marshal drops data: %v", err)
	}

	entity := &Entity{
		Id:   id,
		Type: ServiceTypeDrops,
		Data: jsonData,
	}
	err = db.Create(entity).Error
	if err != nil {
		t.Fatalf("failed to create drops entity: %v", err)
	}
	return id
}

func TestProcessor_GetAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	paged, err := p.AllProvider(model.Page{Number: 1, Size: 250})()
	results := paged.Items
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestProcessor_GetAll_WithData(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	// Create test data
	createLoginEntity(db, t)
	createChannelEntity(db, t)
	createDropsEntity(db, t)

	paged, err := p.AllProvider(model.Page{Number: 1, Size: 250})()
	results := paged.Items
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}
}

func TestProcessor_GetById_LoginService(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	id := createLoginEntity(db, t)

	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}

	loginModel, ok := result.(service.LoginRestModel)
	if !ok {
		t.Fatalf("expected LoginRestModel, got %T", result)
	}

	if loginModel.Id != id.String() {
		t.Errorf("expected id '%s', got '%s'", id.String(), loginModel.Id)
	}
	if len(loginModel.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(loginModel.Tasks))
	}
	if len(loginModel.Tenants) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(loginModel.Tenants))
	}
}

func TestProcessor_GetById_ChannelService(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	id := createChannelEntity(db, t)

	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}

	channelModel, ok := result.(service.ChannelRestModel)
	if !ok {
		t.Fatalf("expected ChannelRestModel, got %T", result)
	}

	if channelModel.Id != id.String() {
		t.Errorf("expected id '%s', got '%s'", id.String(), channelModel.Id)
	}
	if len(channelModel.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(channelModel.Tasks))
	}
	if len(channelModel.Tenants) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(channelModel.Tenants))
	}
}

func TestProcessor_GetById_DropsService(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	id := createDropsEntity(db, t)

	result, err := p.GetById(id)
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}

	dropsModel, ok := result.(service.GenericRestModel)
	if !ok {
		t.Fatalf("expected GenericRestModel, got %T", result)
	}

	if dropsModel.Id != id.String() {
		t.Errorf("expected id '%s', got '%s'", id.String(), dropsModel.Id)
	}
	if len(dropsModel.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(dropsModel.Tasks))
	}
}

func TestProcessor_GetById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	l := testLogger()
	ctx := context.Background()
	p := NewProcessor(l, ctx, db)

	nonExistentId := uuid.New()
	_, err := p.GetById(nonExistentId)
	if err == nil {
		t.Error("expected error for non-existent service")
	}
}

func TestMake_LoginService(t *testing.T) {
	testId := uuid.New()
	loginData := service.LoginRestModel{
		Tasks: []task.RestModel{
			{Type: "heartbeat", Interval: 10000},
		},
		Tenants: []service.LoginTenantRestModel{
			{Id: "tenant-1", Port: 8484},
		},
	}
	jsonData, err := json.Marshal(loginData)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	entity := Entity{
		Id:          testId,
		Type:        ServiceTypeLogin,
		Data:        jsonData,
		Environment: "pr-99",
	}

	result, err := Make(entity)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}

	loginModel, ok := result.(service.LoginRestModel)
	if !ok {
		t.Fatalf("expected LoginRestModel, got %T", result)
	}

	if loginModel.Id != testId.String() {
		t.Errorf("expected id '%s', got '%s'", testId.String(), loginModel.Id)
	}
	// task-48 fix round 2 Critical 1: Make() must copy Entity.Environment
	// onto the RestModel, or cleanup.sh's environment-scoped reclaim filter
	// never matches a real row.
	if loginModel.Environment != "pr-99" {
		t.Errorf("expected environment 'pr-99', got '%s'", loginModel.Environment)
	}
}

func TestMake_ChannelService(t *testing.T) {
	testId := uuid.New()
	channelData := service.ChannelRestModel{
		Tasks: []task.RestModel{
			{Type: "respawn", Interval: 5000},
		},
	}
	jsonData, err := json.Marshal(channelData)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	entity := Entity{
		Id:          testId,
		Type:        ServiceTypeChannel,
		Data:        jsonData,
		Environment: "pr-99",
	}

	result, err := Make(entity)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}

	channelModel, ok := result.(service.ChannelRestModel)
	if !ok {
		t.Fatalf("expected ChannelRestModel, got %T", result)
	}

	if channelModel.Id != testId.String() {
		t.Errorf("expected id '%s', got '%s'", testId.String(), channelModel.Id)
	}
	if channelModel.Environment != "pr-99" {
		t.Errorf("expected environment 'pr-99', got '%s'", channelModel.Environment)
	}
}

func TestMake_DropsService(t *testing.T) {
	testId := uuid.New()
	dropsData := service.GenericRestModel{
		Tasks: []task.RestModel{
			{Type: "cleanup", Interval: 60000},
		},
	}
	jsonData, err := json.Marshal(dropsData)
	if err != nil {
		t.Fatalf("failed to marshal test data: %v", err)
	}

	entity := Entity{
		Id:          testId,
		Type:        ServiceTypeDrops,
		Data:        jsonData,
		Environment: "pr-99",
	}

	result, err := Make(entity)
	if err != nil {
		t.Fatalf("Make failed: %v", err)
	}

	dropsModel, ok := result.(service.GenericRestModel)
	if !ok {
		t.Fatalf("expected GenericRestModel, got %T", result)
	}

	if dropsModel.Id != testId.String() {
		t.Errorf("expected id '%s', got '%s'", testId.String(), dropsModel.Id)
	}
	if dropsModel.Environment != "pr-99" {
		t.Errorf("expected environment 'pr-99', got '%s'", dropsModel.Environment)
	}
}

func TestMake_InvalidServiceType(t *testing.T) {
	entity := Entity{
		Id:   uuid.New(),
		Type: ServiceType("invalid-service"),
		Data: json.RawMessage(`{}`),
	}

	_, err := Make(entity)
	if err == nil {
		t.Error("expected error for invalid service type")
	}
}

func TestMake_InvalidJSON(t *testing.T) {
	entity := Entity{
		Id:   uuid.New(),
		Type: ServiceTypeLogin,
		Data: json.RawMessage(`{invalid json`),
	}

	_, err := Make(entity)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestProcessor_Create_EnqueuesOutboxRow(t *testing.T) {
	t.Setenv(EnvServiceStatusTopic, "test.svc.topic")

	db := setupTestDB(t)
	if err := outboxlib.Migration(db); err != nil {
		t.Fatalf("outbox migration: %v", err)
	}
	p := NewProcessor(testLogger(), context.Background(), db)

	id, err := p.Create(service.InputRestModel{
		Type: string(ServiceTypeChannel),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	var ents []outboxlib.Entity
	if err := db.Find(&ents).Error; err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	if len(ents) != 1 {
		t.Fatalf("want 1 outbox row, got %d", len(ents))
	}
	if got, want := ents[0].Topic, "test.svc.topic"; got != want {
		t.Errorf("topic: got %q want %q", got, want)
	}
	if got, want := string(ents[0].MessageKey), "service:"+id.String(); got != want {
		t.Errorf("key: got %q want %q", got, want)
	}
	if ents[0].MessageValue == nil {
		t.Errorf("envelope value must not be nil on Create")
	}
}

func TestProcessor_Create_NoTopicEnv_SkipsEnqueue(t *testing.T) {
	// EnvServiceStatusTopic intentionally unset.
	db := setupTestDB(t)
	if err := outboxlib.Migration(db); err != nil {
		t.Fatalf("outbox migration: %v", err)
	}
	p := NewProcessor(testLogger(), context.Background(), db)

	if _, err := p.Create(service.InputRestModel{Type: string(ServiceTypeChannel)}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var count int64
	if err := db.Model(&outboxlib.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != 0 {
		t.Errorf("want 0 outbox rows when topic unset, got %d", count)
	}
}

// TestPrEnvironmentCannotPatchAMainOwnedServiceRow is the C2 regression test
// (task-232 FR-7.8, G7): it is what makes "main is never mutated to serve
// an ephemeral environment" VERIFIED rather than merely intended. A pr-123
// caller must be rejected with scope.ErrCrossEnvironmentWrite - not a
// generic not-found - and main's row must be byte-identical afterwards.
func TestPrEnvironmentCannotPatchAMainOwnedServiceRow(t *testing.T) {
	db := testDatabase(t)
	before := seedService(t, db, "main", ServiceTypeLogin)

	err := NewProcessor(testLogger(), envContext(t, "pr-123"), db).
		UpdateById(before.Id, patchWithNewTenant(t))
	if !errors.Is(err, scope.ErrCrossEnvironmentWrite) {
		t.Fatalf("got %v, want ErrCrossEnvironmentWrite", err)
	}

	var after testEntity
	if err := db.First(&after, "id = ?", before.Id).Error; err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if string(after.Data) != string(before.Data) {
		t.Fatalf("main's row changed:\n before %s\n after  %s", before.Data, after.Data)
	}
}
