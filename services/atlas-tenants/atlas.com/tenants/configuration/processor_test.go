package configuration_test

import (
	"atlas-tenants/configuration"
	"atlas-tenants/test"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gorm.io/gorm"
)

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

func createTestRoute(id, name string) map[string]interface{} {
	return map[string]interface{}{
		"type": "routes",
		"id":   id,
		"attributes": map[string]interface{}{
			"name":                   name,
			"startMapId":             float64(100000000),
			"stagingMapId":           float64(100000001),
			"enRouteMapIds":          []interface{}{float64(100000002)},
			"destinationMapId":       float64(100000003),
			"observationMapId":       float64(100000004),
			"boardingWindowDuration": float64(60000),
			"preDepartureDuration":   float64(30000),
			"travelDuration":         float64(120000),
			"cycleInterval":          float64(300000),
		},
	}
}

func createTestVessel(id, name, routeAID, routeBID string) map[string]interface{} {
	return map[string]interface{}{
		"type": "vessels",
		"id":   id,
		"attributes": map[string]interface{}{
			"name":            name,
			"routeAID":        routeAID,
			"routeBID":        routeBID,
			"turnaroundDelay": float64(30000),
		},
	}
}

func (p *testProcessor) createRoute(tenantId uuid.UUID, route map[string]interface{}) (configuration.Model, error) {
	resourceData, err := configuration.CreateSingleRouteJsonData(route)
	if err != nil {
		return configuration.Model{}, err
	}

	entity := configuration.NewEntityBuilder().
		SetID(uuid.New()).
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(resourceData).
		Build()

	if err := configuration.CreateConfiguration(p.db, entity); err != nil {
		return configuration.Model{}, err
	}

	return configuration.Make(entity)
}

func (p *testProcessor) getAllRoutes(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return configuration.GetAllRoutesProvider(tenantId)(p.db)()
}

func (p *testProcessor) getRouteById(tenantId uuid.UUID, routeID string) (map[string]interface{}, error) {
	return configuration.GetRouteByIdProvider(tenantId, routeID)(p.db)()
}

func (p *testProcessor) createVessel(tenantId uuid.UUID, vessel map[string]interface{}) (configuration.Model, error) {
	resourceData, err := configuration.CreateSingleVesselJsonData(vessel)
	if err != nil {
		return configuration.Model{}, err
	}

	entity := configuration.NewEntityBuilder().
		SetID(uuid.New()).
		SetTenantId(tenantId).
		SetResourceName("vessels").
		SetResourceData(resourceData).
		Build()

	if err := configuration.CreateConfiguration(p.db, entity); err != nil {
		return configuration.Model{}, err
	}

	return configuration.Make(entity)
}

func (p *testProcessor) getAllVessels(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return configuration.GetAllVesselsProvider(tenantId)(p.db)()
}

func (p *testProcessor) getVesselById(tenantId uuid.UUID, vesselID string) (map[string]interface{}, error) {
	return configuration.GetVesselByIdProvider(tenantId, vesselID)(p.db)()
}

// Route Tests

func TestCreateRoute_Success(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	route := createTestRoute("route-1", "Test Route")

	m, err := processor.createRoute(tenantId, route)
	if err != nil {
		t.Fatalf("createRoute() unexpected error: %v", err)
	}
	if m.TenantId() != tenantId {
		t.Errorf("m.TenantId() = %v, want %v", m.TenantId(), tenantId)
	}
	if m.ResourceName() != "routes" {
		t.Errorf("m.ResourceName() = %s, want 'routes'", m.ResourceName())
	}
}

func TestGetAllRoutes_Empty(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	_, err := processor.getAllRoutes(tenantId)
	// When no configuration exists, this should error with record not found
	if err == nil {
		t.Error("getAllRoutes() expected error for non-existent configuration")
	}
}

func TestGetAllRoutes_WithRoutes(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	route := createTestRoute("route-1", "Test Route")

	_, err := processor.createRoute(tenantId, route)
	if err != nil {
		t.Fatalf("createRoute() unexpected error: %v", err)
	}

	routes, err := processor.getAllRoutes(tenantId)
	if err != nil {
		t.Fatalf("getAllRoutes() unexpected error: %v", err)
	}
	if len(routes) != 1 {
		t.Errorf("len(routes) = %d, want 1", len(routes))
	}
}

func TestGetRouteById_Found(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	route := createTestRoute("route-1", "Test Route")

	_, err := processor.createRoute(tenantId, route)
	if err != nil {
		t.Fatalf("createRoute() unexpected error: %v", err)
	}

	found, err := processor.getRouteById(tenantId, "route-1")
	if err != nil {
		t.Fatalf("getRouteById() unexpected error: %v", err)
	}
	if found["id"] != "route-1" {
		t.Errorf("found[id] = %v, want 'route-1'", found["id"])
	}
}

func TestGetRouteById_NotFound(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	route := createTestRoute("route-1", "Test Route")

	_, err := processor.createRoute(tenantId, route)
	if err != nil {
		t.Fatalf("createRoute() unexpected error: %v", err)
	}

	_, err = processor.getRouteById(tenantId, "non-existent")
	if err == nil {
		t.Error("getRouteById() expected error for non-existent route")
	}
}

// Vessel Tests

func TestCreateVessel_Success(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	vessel := createTestVessel("vessel-1", "Test Vessel", "route-a", "route-b")

	m, err := processor.createVessel(tenantId, vessel)
	if err != nil {
		t.Fatalf("createVessel() unexpected error: %v", err)
	}
	if m.TenantId() != tenantId {
		t.Errorf("m.TenantId() = %v, want %v", m.TenantId(), tenantId)
	}
	if m.ResourceName() != "vessels" {
		t.Errorf("m.ResourceName() = %s, want 'vessels'", m.ResourceName())
	}
}

func TestGetAllVessels_Empty(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	_, err := processor.getAllVessels(tenantId)
	// When no configuration exists, this should error with record not found
	if err == nil {
		t.Error("getAllVessels() expected error for non-existent configuration")
	}
}

func TestGetAllVessels_WithVessels(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	vessel := createTestVessel("vessel-1", "Test Vessel", "route-a", "route-b")

	_, err := processor.createVessel(tenantId, vessel)
	if err != nil {
		t.Fatalf("createVessel() unexpected error: %v", err)
	}

	vessels, err := processor.getAllVessels(tenantId)
	if err != nil {
		t.Fatalf("getAllVessels() unexpected error: %v", err)
	}
	if len(vessels) != 1 {
		t.Errorf("len(vessels) = %d, want 1", len(vessels))
	}
}

func TestGetVesselById_Found(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	vessel := createTestVessel("vessel-1", "Test Vessel", "route-a", "route-b")

	_, err := processor.createVessel(tenantId, vessel)
	if err != nil {
		t.Fatalf("createVessel() unexpected error: %v", err)
	}

	found, err := processor.getVesselById(tenantId, "vessel-1")
	if err != nil {
		t.Fatalf("getVesselById() unexpected error: %v", err)
	}
	if found["id"] != "vessel-1" {
		t.Errorf("found[id] = %v, want 'vessel-1'", found["id"])
	}
}

func TestGetVesselById_NotFound(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	vessel := createTestVessel("vessel-1", "Test Vessel", "route-a", "route-b")

	_, err := processor.createVessel(tenantId, vessel)
	if err != nil {
		t.Fatalf("createVessel() unexpected error: %v", err)
	}

	_, err = processor.getVesselById(tenantId, "non-existent")
	if err == nil {
		t.Error("getVesselById() expected error for non-existent vessel")
	}
}

// Entity Builder Tests

func TestEntityBuilder(t *testing.T) {
	id := uuid.New()
	tenantId := uuid.New()
	resourceData := json.RawMessage(`{"data": []}`)

	entity := configuration.NewEntityBuilder().
		SetID(id).
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(resourceData).
		Build()

	if entity.ID != id {
		t.Errorf("entity.ID = %v, want %v", entity.ID, id)
	}
	if entity.TenantId != tenantId {
		t.Errorf("entity.TenantId = %v, want %v", entity.TenantId, tenantId)
	}
	if entity.ResourceName != "routes" {
		t.Errorf("entity.ResourceName = %s, want 'routes'", entity.ResourceName)
	}
}

func TestFromModel(t *testing.T) {
	tenantId := uuid.New()
	resourceData := json.RawMessage(`{"data": []}`)

	model, err := configuration.NewModelBuilder().
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(resourceData).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected error: %v", err)
	}

	entity := configuration.FromModel(model)

	if entity.ID != model.ID() {
		t.Errorf("entity.ID = %v, want %v", entity.ID, model.ID())
	}
	if entity.TenantId != model.TenantId() {
		t.Errorf("entity.TenantId = %v, want %v", entity.TenantId, model.TenantId())
	}
	if entity.ResourceName != model.ResourceName() {
		t.Errorf("entity.ResourceName = %s, want %s", entity.ResourceName, model.ResourceName())
	}
}

func TestMake(t *testing.T) {
	id := uuid.New()
	tenantId := uuid.New()
	resourceData := json.RawMessage(`{"data": []}`)

	entity := configuration.NewEntityBuilder().
		SetID(id).
		SetTenantId(tenantId).
		SetResourceName("routes").
		SetResourceData(resourceData).
		Build()

	model, err := configuration.Make(entity)
	if err != nil {
		t.Fatalf("Make() unexpected error: %v", err)
	}

	if model.ID() != id {
		t.Errorf("model.ID() = %v, want %v", model.ID(), id)
	}
	if model.TenantId() != tenantId {
		t.Errorf("model.TenantId() = %v, want %v", model.TenantId(), tenantId)
	}
	if model.ResourceName() != "routes" {
		t.Errorf("model.ResourceName() = %s, want 'routes'", model.ResourceName())
	}
}

// REST Transform Tests

func TestTransformRoute(t *testing.T) {
	route := createTestRoute("route-1", "Test Route")

	restModel, err := configuration.TransformRoute(route)
	if err != nil {
		t.Fatalf("TransformRoute() unexpected error: %v", err)
	}
	if restModel.Id != "route-1" {
		t.Errorf("restModel.Id = %s, want 'route-1'", restModel.Id)
	}
	if restModel.Name != "Test Route" {
		t.Errorf("restModel.Name = %s, want 'Test Route'", restModel.Name)
	}
	if restModel.StartMapId != 100000000 {
		t.Errorf("restModel.StartMapId = %d, want 100000000", restModel.StartMapId)
	}
}

func TestTransformVessel(t *testing.T) {
	vessel := createTestVessel("vessel-1", "Test Vessel", "route-a", "route-b")

	restModel, err := configuration.TransformVessel(vessel)
	if err != nil {
		t.Fatalf("TransformVessel() unexpected error: %v", err)
	}
	if restModel.Id != "vessel-1" {
		t.Errorf("restModel.Id = %s, want 'vessel-1'", restModel.Id)
	}
	if restModel.Name != "Test Vessel" {
		t.Errorf("restModel.Name = %s, want 'Test Vessel'", restModel.Name)
	}
	if restModel.RouteAID != "route-a" {
		t.Errorf("restModel.RouteAID = %s, want 'route-a'", restModel.RouteAID)
	}
}

func TestExtractRoute(t *testing.T) {
	restModel := configuration.RouteRestModel{
		Id:                     "route-1",
		Name:                   "Test Route",
		StartMapId:             100000000,
		StagingMapId:           100000001,
		EnRouteMapIds:          []uint32{100000002},
		DestinationMapId:       100000003,
		ObservationMapId:       100000004,
		BoardingWindowDuration: 60000,
		PreDepartureDuration:   30000,
		TravelDuration:         120000,
		CycleInterval:          300000,
	}

	route, err := configuration.ExtractRoute(restModel)
	if err != nil {
		t.Fatalf("ExtractRoute() unexpected error: %v", err)
	}
	if route["id"] != "route-1" {
		t.Errorf("route[id] = %v, want 'route-1'", route["id"])
	}
	if route["type"] != "routes" {
		t.Errorf("route[type] = %v, want 'routes'", route["type"])
	}
}

func TestExtractVessel(t *testing.T) {
	restModel := configuration.VesselRestModel{
		Id:              "vessel-1",
		Name:            "Test Vessel",
		RouteAID:        "route-a",
		RouteBID:        "route-b",
		TurnaroundDelay: 30000,
	}

	vessel, err := configuration.ExtractVessel(restModel)
	if err != nil {
		t.Fatalf("ExtractVessel() unexpected error: %v", err)
	}
	if vessel["id"] != "vessel-1" {
		t.Errorf("vessel[id] = %v, want 'vessel-1'", vessel["id"])
	}
	if vessel["type"] != "vessels" {
		t.Errorf("vessel[type] = %v, want 'vessels'", vessel["type"])
	}
}

func TestCreateRouteJsonData(t *testing.T) {
	routes := []map[string]interface{}{
		createTestRoute("route-1", "Route 1"),
		createTestRoute("route-2", "Route 2"),
	}

	data, err := configuration.CreateRouteJsonData(routes)
	if err != nil {
		t.Fatalf("CreateRouteJsonData() unexpected error: %v", err)
	}
	if data == nil {
		t.Error("CreateRouteJsonData() returned nil")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("CreateRouteJsonData() produced invalid JSON: %v", err)
	}
	if parsed["data"] == nil {
		t.Error("CreateRouteJsonData() should have 'data' field")
	}
}

func TestCreateSingleRouteJsonData(t *testing.T) {
	route := createTestRoute("route-1", "Test Route")

	data, err := configuration.CreateSingleRouteJsonData(route)
	if err != nil {
		t.Fatalf("CreateSingleRouteJsonData() unexpected error: %v", err)
	}
	if data == nil {
		t.Error("CreateSingleRouteJsonData() returned nil")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("CreateSingleRouteJsonData() produced invalid JSON: %v", err)
	}
}

// Tenant Isolation Tests

func TestTenantIsolation_Routes(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create route for tenant 1
	route1 := createTestRoute("route-t1", "Tenant 1 Route")
	_, err := processor.createRoute(tenant1, route1)
	if err != nil {
		t.Fatalf("createRoute() for tenant 1 unexpected error: %v", err)
	}

	// Create route for tenant 2
	route2 := createTestRoute("route-t2", "Tenant 2 Route")
	_, err = processor.createRoute(tenant2, route2)
	if err != nil {
		t.Fatalf("createRoute() for tenant 2 unexpected error: %v", err)
	}

	// Tenant 1 should only see their route
	routes1, err := processor.getAllRoutes(tenant1)
	if err != nil {
		t.Fatalf("getAllRoutes() for tenant 1 unexpected error: %v", err)
	}
	if len(routes1) != 1 {
		t.Errorf("tenant 1 routes = %d, want 1", len(routes1))
	}
	if routes1[0]["id"] != "route-t1" {
		t.Errorf("tenant 1 route id = %v, want 'route-t1'", routes1[0]["id"])
	}

	// Tenant 2 should only see their route
	routes2, err := processor.getAllRoutes(tenant2)
	if err != nil {
		t.Fatalf("getAllRoutes() for tenant 2 unexpected error: %v", err)
	}
	if len(routes2) != 1 {
		t.Errorf("tenant 2 routes = %d, want 1", len(routes2))
	}
	if routes2[0]["id"] != "route-t2" {
		t.Errorf("tenant 2 route id = %v, want 'route-t2'", routes2[0]["id"])
	}

	// Tenant 1 should not be able to access tenant 2's route
	_, err = processor.getRouteById(tenant1, "route-t2")
	if err == nil {
		t.Error("tenant 1 should not be able to access tenant 2's route")
	}
}

func TestTenantIsolation_Vessels(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create vessel for tenant 1
	vessel1 := createTestVessel("vessel-t1", "Tenant 1 Vessel", "route-a", "route-b")
	_, err := processor.createVessel(tenant1, vessel1)
	if err != nil {
		t.Fatalf("createVessel() for tenant 1 unexpected error: %v", err)
	}

	// Create vessel for tenant 2
	vessel2 := createTestVessel("vessel-t2", "Tenant 2 Vessel", "route-c", "route-d")
	_, err = processor.createVessel(tenant2, vessel2)
	if err != nil {
		t.Fatalf("createVessel() for tenant 2 unexpected error: %v", err)
	}

	// Tenant 1 should only see their vessel
	vessels1, err := processor.getAllVessels(tenant1)
	if err != nil {
		t.Fatalf("getAllVessels() for tenant 1 unexpected error: %v", err)
	}
	if len(vessels1) != 1 {
		t.Errorf("tenant 1 vessels = %d, want 1", len(vessels1))
	}

	// Tenant 2 should only see their vessel
	vessels2, err := processor.getAllVessels(tenant2)
	if err != nil {
		t.Fatalf("getAllVessels() for tenant 2 unexpected error: %v", err)
	}
	if len(vessels2) != 1 {
		t.Errorf("tenant 2 vessels = %d, want 1", len(vessels2))
	}
}

// REST Model Interface Tests

func TestRouteRestModel_GetID(t *testing.T) {
	r := configuration.RouteRestModel{Id: "test-id"}
	if r.GetID() != "test-id" {
		t.Errorf("GetID() = %s, want 'test-id'", r.GetID())
	}
}

func TestRouteRestModel_SetID(t *testing.T) {
	r := configuration.RouteRestModel{}
	err := r.SetID("new-id")
	if err != nil {
		t.Fatalf("SetID() unexpected error: %v", err)
	}
	if r.Id != "new-id" {
		t.Errorf("r.Id = %s, want 'new-id'", r.Id)
	}
}

func TestRouteRestModel_GetName(t *testing.T) {
	r := configuration.RouteRestModel{}
	if r.GetName() != "routes" {
		t.Errorf("GetName() = %s, want 'routes'", r.GetName())
	}
}

func TestVesselRestModel_GetID(t *testing.T) {
	v := configuration.VesselRestModel{Id: "test-id"}
	if v.GetID() != "test-id" {
		t.Errorf("GetID() = %s, want 'test-id'", v.GetID())
	}
}

func TestVesselRestModel_SetID(t *testing.T) {
	v := configuration.VesselRestModel{}
	err := v.SetID("new-id")
	if err != nil {
		t.Fatalf("SetID() unexpected error: %v", err)
	}
	if v.Id != "new-id" {
		t.Errorf("v.Id = %s, want 'new-id'", v.Id)
	}
}

func TestVesselRestModel_GetName(t *testing.T) {
	v := configuration.VesselRestModel{}
	if v.GetName() != "vessels" {
		t.Errorf("GetName() = %s, want 'vessels'", v.GetName())
	}
}
