package tenant_test

import (
	"atlas-tenants/kafka/message"
	"atlas-tenants/tenant"
	"atlas-tenants/test"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"
)

// testProcessor wraps tenant.ProcessorImpl for testing
// We use the base methods (Create, Update, Delete with buffer) directly
// to avoid needing Kafka producer setup
type testProcessor struct {
	db *gorm.DB
	l  logrus.FieldLogger
}

func setupTestProcessor(t *testing.T) (*testProcessor, func()) {
	db := test.SetupTestDB(t)
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	cleanup := func() {
		test.CleanupTestDB(db)
	}

	return &testProcessor{db: db, l: logger}, cleanup
}

func (p *testProcessor) create(name, region string, majorVersion, minorVersion uint16) (tenant.Model, error) {
	mb := message.NewBuffer()

	m, err := tenant.NewModelBuilder().
		SetName(name).
		SetRegion(region).
		SetMajorVersion(majorVersion).
		SetMinorVersion(minorVersion).
		Build()
	if err != nil {
		return tenant.Model{}, err
	}

	e := tenant.FromModel(m)
	err = tenant.CreateTenant(p.db, e)
	if err != nil {
		return tenant.Model{}, err
	}

	_ = mb // Events would be added here in real processor
	return m, nil
}

func (p *testProcessor) getById(id uuid.UUID) (tenant.Model, error) {
	provider := tenant.GetByIdProvider(id)(p.db)
	e, err := provider()
	if err != nil {
		return tenant.Model{}, err
	}
	return tenant.Make(e)
}

func (p *testProcessor) getAll() ([]tenant.Model, error) {
	provider := tenant.GetAllProvider()(p.db)
	entities, err := provider()
	if err != nil {
		return nil, err
	}
	models := make([]tenant.Model, 0, len(entities))
	for _, e := range entities {
		m, err := tenant.Make(e)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, nil
}

func (p *testProcessor) update(id uuid.UUID, name, region string, majorVersion, minorVersion uint16) (tenant.Model, error) {
	provider := tenant.GetByIdProvider(id)(p.db)
	e, err := provider()
	if err != nil {
		return tenant.Model{}, err
	}

	e.Name = name
	e.Region = region
	e.MajorVersion = majorVersion
	e.MinorVersion = minorVersion

	err = tenant.UpdateTenant(p.db, e)
	if err != nil {
		return tenant.Model{}, err
	}

	return tenant.Make(e)
}

func (p *testProcessor) delete(id uuid.UUID) error {
	return tenant.DeleteTenant(p.db, id)
}

func TestCreate_Success(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	m, err := processor.create("Test Tenant", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create() unexpected error: %v", err)
	}
	if m.Name() != "Test Tenant" {
		t.Errorf("m.Name() = %s, want 'Test Tenant'", m.Name())
	}
	if m.Region() != "GMS" {
		t.Errorf("m.Region() = %s, want 'GMS'", m.Region())
	}
	if m.MajorVersion() != 83 {
		t.Errorf("m.MajorVersion() = %d, want 83", m.MajorVersion())
	}
	if m.MinorVersion() != 1 {
		t.Errorf("m.MinorVersion() = %d, want 1", m.MinorVersion())
	}
	if m.Id() == uuid.Nil {
		t.Error("m.Id() should not be nil UUID")
	}
}

func TestCreate_ValidationError(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	// Missing name should fail validation
	_, err := processor.create("", "GMS", 83, 1)
	if err == nil {
		t.Error("create() expected validation error for missing name")
	}
}

func TestGetById_Found(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	created, err := processor.create("Test Tenant", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create() unexpected error: %v", err)
	}

	found, err := processor.getById(created.Id())
	if err != nil {
		t.Fatalf("getById() unexpected error: %v", err)
	}
	if found.Id() != created.Id() {
		t.Errorf("found.Id() = %v, want %v", found.Id(), created.Id())
	}
	if found.Name() != "Test Tenant" {
		t.Errorf("found.Name() = %s, want 'Test Tenant'", found.Name())
	}
}

func TestGetById_NotFound(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	_, err := processor.getById(uuid.New())
	if err == nil {
		t.Error("getById() expected error for non-existent tenant")
	}
}

func TestGetAll_Empty(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenants, err := processor.getAll()
	if err != nil {
		t.Fatalf("getAll() unexpected error: %v", err)
	}
	if len(tenants) != 0 {
		t.Errorf("len(tenants) = %d, want 0", len(tenants))
	}
}

func TestGetAll_WithTenants(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	// Create some tenants
	_, err := processor.create("Tenant 1", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create() unexpected error: %v", err)
	}
	_, err = processor.create("Tenant 2", "EMS", 90, 2)
	if err != nil {
		t.Fatalf("create() unexpected error: %v", err)
	}

	tenants, err := processor.getAll()
	if err != nil {
		t.Fatalf("getAll() unexpected error: %v", err)
	}
	if len(tenants) != 2 {
		t.Errorf("len(tenants) = %d, want 2", len(tenants))
	}
}

func TestUpdate_Success(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	created, err := processor.create("Original Name", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create() unexpected error: %v", err)
	}

	updated, err := processor.update(created.Id(), "Updated Name", "EMS", 90, 2)
	if err != nil {
		t.Fatalf("update() unexpected error: %v", err)
	}
	if updated.Name() != "Updated Name" {
		t.Errorf("updated.Name() = %s, want 'Updated Name'", updated.Name())
	}
	if updated.Region() != "EMS" {
		t.Errorf("updated.Region() = %s, want 'EMS'", updated.Region())
	}
	if updated.MajorVersion() != 90 {
		t.Errorf("updated.MajorVersion() = %d, want 90", updated.MajorVersion())
	}
	if updated.MinorVersion() != 2 {
		t.Errorf("updated.MinorVersion() = %d, want 2", updated.MinorVersion())
	}
}

func TestUpdate_NotFound(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	_, err := processor.update(uuid.New(), "Name", "GMS", 83, 1)
	if err == nil {
		t.Error("update() expected error for non-existent tenant")
	}
}

func TestDelete_Success(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	created, err := processor.create("Test Tenant", "GMS", 83, 1)
	if err != nil {
		t.Fatalf("create() unexpected error: %v", err)
	}

	err = processor.delete(created.Id())
	if err != nil {
		t.Fatalf("delete() unexpected error: %v", err)
	}

	// Verify tenant is deleted
	_, err = processor.getById(created.Id())
	if err == nil {
		t.Error("getById() expected error for deleted tenant")
	}
}

func TestDelete_NotFound(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	err := processor.delete(uuid.New())
	if err == nil {
		t.Error("delete() expected error for non-existent tenant")
	}
}

func TestMultipleTenants(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	// Create multiple tenants
	tenant1, _ := processor.create("Tenant 1", "GMS", 83, 1)
	tenant2, _ := processor.create("Tenant 2", "EMS", 90, 2)
	tenant3, _ := processor.create("Tenant 3", "JMS", 95, 3)

	// Verify all can be retrieved
	tenants, err := processor.getAll()
	if err != nil {
		t.Fatalf("getAll() unexpected error: %v", err)
	}
	if len(tenants) != 3 {
		t.Errorf("len(tenants) = %d, want 3", len(tenants))
	}

	// Delete middle tenant
	err = processor.delete(tenant2.Id())
	if err != nil {
		t.Fatalf("delete() unexpected error: %v", err)
	}

	// Verify only 2 remain
	tenants, err = processor.getAll()
	if err != nil {
		t.Fatalf("getAll() unexpected error: %v", err)
	}
	if len(tenants) != 2 {
		t.Errorf("len(tenants) = %d, want 2", len(tenants))
	}

	// Verify remaining tenants are correct
	found1, _ := processor.getById(tenant1.Id())
	found3, _ := processor.getById(tenant3.Id())
	if found1.Name() != "Tenant 1" {
		t.Errorf("found1.Name() = %s, want 'Tenant 1'", found1.Name())
	}
	if found3.Name() != "Tenant 3" {
		t.Errorf("found3.Name() = %s, want 'Tenant 3'", found3.Name())
	}
}

func TestEntityBuilder(t *testing.T) {
	id := uuid.New()

	entity := tenant.NewEntityBuilder().
		SetId(id).
		SetName("Test Tenant").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()

	if entity.ID != id {
		t.Errorf("entity.ID = %v, want %v", entity.ID, id)
	}
	if entity.Name != "Test Tenant" {
		t.Errorf("entity.Name = %s, want 'Test Tenant'", entity.Name)
	}
	if entity.Region != "GMS" {
		t.Errorf("entity.Region = %s, want 'GMS'", entity.Region)
	}
}

func TestFromModel(t *testing.T) {
	model, err := tenant.NewModelBuilder().
		SetName("Test Tenant").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	entity := tenant.FromModel(model)

	if entity.ID != model.Id() {
		t.Errorf("entity.ID = %v, want %v", entity.ID, model.Id())
	}
	if entity.Name != model.Name() {
		t.Errorf("entity.Name = %s, want %s", entity.Name, model.Name())
	}
	if entity.Region != model.Region() {
		t.Errorf("entity.Region = %s, want %s", entity.Region, model.Region())
	}
}

func TestMake(t *testing.T) {
	id := uuid.New()
	entity := tenant.NewEntityBuilder().
		SetId(id).
		SetName("Test Tenant").
		SetRegion("GMS").
		SetMajorVersion(83).
		SetMinorVersion(1).
		Build()

	model, err := tenant.Make(entity)
	if err != nil {
		t.Fatalf("Make() unexpected error: %v", err)
	}

	if model.Id() != id {
		t.Errorf("model.Id() = %v, want %v", model.Id(), id)
	}
	if model.Name() != "Test Tenant" {
		t.Errorf("model.Name() = %s, want 'Test Tenant'", model.Name())
	}
	if model.Region() != "GMS" {
		t.Errorf("model.Region() = %s, want 'GMS'", model.Region())
	}
}
