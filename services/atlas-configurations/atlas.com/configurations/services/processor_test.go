package services

import (
	"atlas-configurations/services/service"
	"atlas-configurations/services/task"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testEntity is a SQLite-compatible version of Entity for testing
type testEntity struct {
	Id   uuid.UUID       `gorm:"type:text;primaryKey"`
	Type ServiceType     `gorm:"type:varchar"`
	Data json.RawMessage `gorm:"type:text;not null"`
}

func (testEntity) TableName() string {
	return "services"
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Use SQLite-compatible schema
	err = db.AutoMigrate(&testEntity{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	return db
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

	results, err := p.GetAll()
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

	results, err := p.GetAll()
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
		Id:   testId,
		Type: ServiceTypeLogin,
		Data: jsonData,
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
		Id:   testId,
		Type: ServiceTypeChannel,
		Data: jsonData,
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
		Id:   testId,
		Type: ServiceTypeDrops,
		Data: jsonData,
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
