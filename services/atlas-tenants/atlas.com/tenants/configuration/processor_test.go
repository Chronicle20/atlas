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

func createTestMtsConfig(id string) map[string]interface{} {
	return map[string]interface{}{
		"type": "mts-configs",
		"id":   id,
		"attributes": map[string]interface{}{
			"listingFee":        float64(5000),
			"commissionRate":    float64(0.10),
			"maxActiveListings": float64(10),
			"minLevel":          float64(10),
			"auctionMinHours":   float64(24),
			"auctionMaxHours":   float64(168),
			"priceFloor":        float64(110),
			"pageSize":          float64(16),
			"minBidIncrement":   float64(1),
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

func createTestIncubatorReward(id string, itemId, quantity, weight uint32) map[string]interface{} {
	return map[string]interface{}{
		"type": "incubator-rewards",
		"id":   id,
		"attributes": map[string]interface{}{
			"itemId":   float64(itemId),
			"quantity": float64(quantity),
			"weight":   float64(weight),
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

func (p *testProcessor) createIncubatorReward(tenantId uuid.UUID, reward map[string]interface{}) (configuration.Model, error) {
	resourceData, err := configuration.CreateSingleIncubatorRewardJsonData(reward)
	if err != nil {
		return configuration.Model{}, err
	}

	entity := configuration.NewEntityBuilder().
		SetID(uuid.New()).
		SetTenantId(tenantId).
		SetResourceName("incubator-rewards").
		SetResourceData(resourceData).
		Build()

	if err := configuration.CreateConfiguration(p.db, entity); err != nil {
		return configuration.Model{}, err
	}

	return configuration.Make(entity)
}

func (p *testProcessor) getAllIncubatorRewards(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return configuration.GetAllIncubatorRewardsProvider(tenantId)(p.db)()
}

func (p *testProcessor) getIncubatorRewardById(tenantId uuid.UUID, rewardID string) (map[string]interface{}, error) {
	return configuration.GetIncubatorRewardByIdProvider(tenantId, rewardID)(p.db)()
}

func (p *testProcessor) createMtsConfig(tenantId uuid.UUID, cfg map[string]interface{}) (configuration.Model, error) {
	resourceData, err := configuration.CreateSingleMtsConfigJsonData(cfg)
	if err != nil {
		return configuration.Model{}, err
	}

	entity := configuration.NewEntityBuilder().
		SetID(uuid.New()).
		SetTenantId(tenantId).
		SetResourceName("mts-configs").
		SetResourceData(resourceData).
		Build()

	if err := configuration.CreateConfiguration(p.db, entity); err != nil {
		return configuration.Model{}, err
	}

	return configuration.Make(entity)
}

func (p *testProcessor) getAllMtsConfigs(tenantId uuid.UUID) ([]map[string]interface{}, error) {
	return configuration.GetAllMtsConfigsProvider(tenantId)(p.db)()
}

func (p *testProcessor) getMtsConfigById(tenantId uuid.UUID, configID string) (map[string]interface{}, error) {
	return configuration.GetMtsConfigByIdProvider(tenantId, configID)(p.db)()
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

// IncubatorReward Tests

func TestCreateIncubatorReward_Success(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	reward := createTestIncubatorReward("red-potion", 2000000, 50, 40)

	m, err := processor.createIncubatorReward(tenantId, reward)
	if err != nil {
		t.Fatalf("createIncubatorReward() unexpected error: %v", err)
	}
	if m.TenantId() != tenantId {
		t.Errorf("m.TenantId() = %v, want %v", m.TenantId(), tenantId)
	}
	if m.ResourceName() != "incubator-rewards" {
		t.Errorf("m.ResourceName() = %s, want 'incubator-rewards'", m.ResourceName())
	}
}

func TestGetAllIncubatorRewards_Empty(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	_, err := processor.getAllIncubatorRewards(tenantId)
	// When no configuration exists, this should error with record not found
	if err == nil {
		t.Error("getAllIncubatorRewards() expected error for non-existent configuration")
	}
}

func TestGetAllIncubatorRewards_WithIncubatorRewards(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	reward := createTestIncubatorReward("red-potion", 2000000, 50, 40)

	_, err := processor.createIncubatorReward(tenantId, reward)
	if err != nil {
		t.Fatalf("createIncubatorReward() unexpected error: %v", err)
	}

	rewards, err := processor.getAllIncubatorRewards(tenantId)
	if err != nil {
		t.Fatalf("getAllIncubatorRewards() unexpected error: %v", err)
	}
	if len(rewards) != 1 {
		t.Errorf("len(rewards) = %d, want 1", len(rewards))
	}

	attrs, ok := rewards[0]["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("rewards[0][attributes] not a map")
	}
	if attrs["itemId"] != float64(2000000) {
		t.Errorf("attrs[itemId] = %v, want 2000000", attrs["itemId"])
	}
}

func TestGetIncubatorRewardById_Found(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	reward := createTestIncubatorReward("red-potion", 2000000, 50, 40)

	_, err := processor.createIncubatorReward(tenantId, reward)
	if err != nil {
		t.Fatalf("createIncubatorReward() unexpected error: %v", err)
	}

	found, err := processor.getIncubatorRewardById(tenantId, "red-potion")
	if err != nil {
		t.Fatalf("getIncubatorRewardById() unexpected error: %v", err)
	}
	if found["id"] != "red-potion" {
		t.Errorf("found[id] = %v, want 'red-potion'", found["id"])
	}
}

func TestGetIncubatorRewardById_NotFound(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	reward := createTestIncubatorReward("red-potion", 2000000, 50, 40)

	_, err := processor.createIncubatorReward(tenantId, reward)
	if err != nil {
		t.Fatalf("createIncubatorReward() unexpected error: %v", err)
	}

	_, err = processor.getIncubatorRewardById(tenantId, "non-existent")
	if err == nil {
		t.Error("getIncubatorRewardById() expected error for non-existent incubator reward")
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

func TestTransformIncubatorReward(t *testing.T) {
	reward := createTestIncubatorReward("red-potion", 2000000, 50, 40)

	restModel, err := configuration.TransformIncubatorReward(reward)
	if err != nil {
		t.Fatalf("TransformIncubatorReward() unexpected error: %v", err)
	}
	if restModel.Id != "red-potion" {
		t.Errorf("restModel.Id = %s, want 'red-potion'", restModel.Id)
	}
	if restModel.ItemId != 2000000 {
		t.Errorf("restModel.ItemId = %d, want 2000000", restModel.ItemId)
	}
	if restModel.Quantity != 50 {
		t.Errorf("restModel.Quantity = %d, want 50", restModel.Quantity)
	}
	if restModel.Weight != 40 {
		t.Errorf("restModel.Weight = %d, want 40", restModel.Weight)
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

func TestExtractIncubatorReward(t *testing.T) {
	restModel := configuration.IncubatorRewardRestModel{
		Id:       "red-potion",
		ItemId:   2000000,
		Quantity: 50,
		Weight:   40,
		EggId:    4170000,
	}

	reward, err := configuration.ExtractIncubatorReward(restModel)
	if err != nil {
		t.Fatalf("ExtractIncubatorReward() unexpected error: %v", err)
	}
	if reward["id"] != "red-potion" {
		t.Errorf("reward[id] = %v, want 'red-potion'", reward["id"])
	}
	if reward["type"] != "incubator-rewards" {
		t.Errorf("reward[type] = %v, want 'incubator-rewards'", reward["type"])
	}
	attrs, ok := reward["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("reward[attributes] not a map")
	}
	if attrs["eggId"] != uint32(4170000) {
		t.Errorf("attrs[eggId] = %v, want 4170000", attrs["eggId"])
	}
}

func TestIncubatorRewardTransformExtractRoundTrip(t *testing.T) {
	reward := createTestIncubatorReward("red-potion", 2000000, 50, 40)

	restModel, err := configuration.TransformIncubatorReward(reward)
	if err != nil {
		t.Fatalf("TransformIncubatorReward() unexpected error: %v", err)
	}

	extracted, err := configuration.ExtractIncubatorReward(restModel)
	if err != nil {
		t.Fatalf("ExtractIncubatorReward() unexpected error: %v", err)
	}

	attrs, ok := extracted["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("extracted[attributes] not a map")
	}
	if attrs["itemId"] != uint32(2000000) {
		t.Errorf("attrs[itemId] = %v, want 2000000", attrs["itemId"])
	}
	if attrs["quantity"] != uint32(50) {
		t.Errorf("attrs[quantity] = %v, want 50", attrs["quantity"])
	}
	if attrs["weight"] != uint32(40) {
		t.Errorf("attrs[weight] = %v, want 40", attrs["weight"])
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

func TestTenantIsolation_IncubatorRewards(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	// Create incubator reward for tenant 1
	reward1 := createTestIncubatorReward("reward-t1", 2000000, 50, 40)
	_, err := processor.createIncubatorReward(tenant1, reward1)
	if err != nil {
		t.Fatalf("createIncubatorReward() for tenant 1 unexpected error: %v", err)
	}

	// Create incubator reward for tenant 2
	reward2 := createTestIncubatorReward("reward-t2", 2000001, 50, 30)
	_, err = processor.createIncubatorReward(tenant2, reward2)
	if err != nil {
		t.Fatalf("createIncubatorReward() for tenant 2 unexpected error: %v", err)
	}

	// Tenant 1 should only see their incubator reward
	rewards1, err := processor.getAllIncubatorRewards(tenant1)
	if err != nil {
		t.Fatalf("getAllIncubatorRewards() for tenant 1 unexpected error: %v", err)
	}
	if len(rewards1) != 1 {
		t.Errorf("tenant 1 incubator rewards = %d, want 1", len(rewards1))
	}

	// Tenant 2 should only see their incubator reward
	rewards2, err := processor.getAllIncubatorRewards(tenant2)
	if err != nil {
		t.Fatalf("getAllIncubatorRewards() for tenant 2 unexpected error: %v", err)
	}
	if len(rewards2) != 1 {
		t.Errorf("tenant 2 incubator rewards = %d, want 1", len(rewards2))
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

func TestIncubatorRewardRestModel_GetID(t *testing.T) {
	v := configuration.IncubatorRewardRestModel{Id: "test-id"}
	if v.GetID() != "test-id" {
		t.Errorf("GetID() = %s, want 'test-id'", v.GetID())
	}
}

func TestIncubatorRewardRestModel_SetID(t *testing.T) {
	v := configuration.IncubatorRewardRestModel{}
	err := v.SetID("new-id")
	if err != nil {
		t.Fatalf("SetID() unexpected error: %v", err)
	}
	if v.Id != "new-id" {
		t.Errorf("v.Id = %s, want 'new-id'", v.Id)
	}
}

func TestIncubatorRewardRestModel_GetName(t *testing.T) {
	v := configuration.IncubatorRewardRestModel{}
	if v.GetName() != "incubator-rewards" {
		t.Errorf("GetName() = %s, want 'incubator-rewards'", v.GetName())
	}
}

// MTS Config Tests

func TestCreateMtsConfig_Success(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	cfg := createTestMtsConfig("mts-config-1")

	m, err := processor.createMtsConfig(tenantId, cfg)
	if err != nil {
		t.Fatalf("createMtsConfig() unexpected error: %v", err)
	}
	if m.TenantId() != tenantId {
		t.Errorf("m.TenantId() = %v, want %v", m.TenantId(), tenantId)
	}
	if m.ResourceName() != "mts-configs" {
		t.Errorf("m.ResourceName() = %s, want 'mts-configs'", m.ResourceName())
	}
}

func TestGetAllMtsConfigs_Empty(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	_, err := processor.getAllMtsConfigs(tenantId)
	if err == nil {
		t.Error("getAllMtsConfigs() expected error for non-existent configuration")
	}
}

func TestGetMtsConfigById_Found(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	cfg := createTestMtsConfig("mts-config-1")

	_, err := processor.createMtsConfig(tenantId, cfg)
	if err != nil {
		t.Fatalf("createMtsConfig() unexpected error: %v", err)
	}

	found, err := processor.getMtsConfigById(tenantId, "mts-config-1")
	if err != nil {
		t.Fatalf("getMtsConfigById() unexpected error: %v", err)
	}
	if found["id"] != "mts-config-1" {
		t.Errorf("found[id] = %v, want 'mts-config-1'", found["id"])
	}
}

func TestGetMtsConfigById_NotFound(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	cfg := createTestMtsConfig("mts-config-1")

	_, err := processor.createMtsConfig(tenantId, cfg)
	if err != nil {
		t.Fatalf("createMtsConfig() unexpected error: %v", err)
	}

	_, err = processor.getMtsConfigById(tenantId, "non-existent")
	if err == nil {
		t.Error("getMtsConfigById() expected error for non-existent config")
	}
}

func TestTransformMtsConfig(t *testing.T) {
	cfg := createTestMtsConfig("mts-config-1")

	restModel, err := configuration.TransformMtsConfig(cfg)
	if err != nil {
		t.Fatalf("TransformMtsConfig() unexpected error: %v", err)
	}
	if restModel.Id != "mts-config-1" {
		t.Errorf("restModel.Id = %s, want 'mts-config-1'", restModel.Id)
	}
	if restModel.ListingFee != 5000 {
		t.Errorf("restModel.ListingFee = %d, want 5000", restModel.ListingFee)
	}
	if restModel.CommissionRate != 0.10 {
		t.Errorf("restModel.CommissionRate = %v, want 0.10", restModel.CommissionRate)
	}
	if restModel.MaxActiveListings != 10 {
		t.Errorf("restModel.MaxActiveListings = %d, want 10", restModel.MaxActiveListings)
	}
	if restModel.MinLevel != 10 {
		t.Errorf("restModel.MinLevel = %d, want 10", restModel.MinLevel)
	}
	if restModel.AuctionMinHours != 24 {
		t.Errorf("restModel.AuctionMinHours = %d, want 24", restModel.AuctionMinHours)
	}
	if restModel.AuctionMaxHours != 168 {
		t.Errorf("restModel.AuctionMaxHours = %d, want 168", restModel.AuctionMaxHours)
	}
	if restModel.PriceFloor != 110 {
		t.Errorf("restModel.PriceFloor = %d, want 110", restModel.PriceFloor)
	}
	if restModel.PageSize != 16 {
		t.Errorf("restModel.PageSize = %d, want 16", restModel.PageSize)
	}
	if restModel.MinBidIncrement != 1 {
		t.Errorf("restModel.MinBidIncrement = %d, want 1", restModel.MinBidIncrement)
	}
}

func TestExtractMtsConfig(t *testing.T) {
	restModel := configuration.MtsConfigRestModel{
		Id:                "mts-config-1",
		ListingFee:        5000,
		CommissionRate:    0.10,
		MaxActiveListings: 10,
		MinLevel:          10,
		AuctionMinHours:   24,
		AuctionMaxHours:   168,
		PriceFloor:        110,
		PageSize:          16,
		MinBidIncrement:   1,
	}

	cfg, err := configuration.ExtractMtsConfig(restModel)
	if err != nil {
		t.Fatalf("ExtractMtsConfig() unexpected error: %v", err)
	}
	if cfg["id"] != "mts-config-1" {
		t.Errorf("cfg[id] = %v, want 'mts-config-1'", cfg["id"])
	}
	if cfg["type"] != "mts-configs" {
		t.Errorf("cfg[type] = %v, want 'mts-configs'", cfg["type"])
	}
}

func TestMtsConfigRestModel_GetName(t *testing.T) {
	m := configuration.MtsConfigRestModel{}
	if m.GetName() != "mts-configs" {
		t.Errorf("GetName() = %s, want 'mts-configs'", m.GetName())
	}
}

func TestMtsConfigRoundTrip(t *testing.T) {
	processor, cleanup := setupTestProcessor(t)
	defer cleanup()

	tenantId := uuid.New()
	cfg := createTestMtsConfig("mts-config-1")

	_, err := processor.createMtsConfig(tenantId, cfg)
	if err != nil {
		t.Fatalf("createMtsConfig() unexpected error: %v", err)
	}

	found, err := processor.getMtsConfigById(tenantId, "mts-config-1")
	if err != nil {
		t.Fatalf("getMtsConfigById() unexpected error: %v", err)
	}

	rm, err := configuration.TransformMtsConfig(found)
	if err != nil {
		t.Fatalf("TransformMtsConfig() unexpected error: %v", err)
	}
	if rm.ListingFee != 5000 {
		t.Errorf("round-trip ListingFee = %d, want 5000", rm.ListingFee)
	}
	if rm.PriceFloor != 110 {
		t.Errorf("round-trip PriceFloor = %d, want 110", rm.PriceFloor)
	}
}
