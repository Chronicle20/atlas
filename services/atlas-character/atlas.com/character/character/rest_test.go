package character_test

import (
	"atlas-character/character"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/jtumidanski/api2go/jsonapi"
	goredis "github.com/redis/go-redis/v9"
)

func testTenantContext() context.Context {
	return tenant.WithContext(context.Background(), testTenant())
}

type Server struct {
	baseUrl string
	prefix  string
}

func (s Server) GetBaseURL() string {
	return s.baseUrl
}

func (s Server) GetPrefix() string {
	return s.prefix
}

func GetServer() Server {
	return Server{
		baseUrl: "",
		prefix:  "/api/",
	}
}

func setupTestRedis(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func TestMarshalUnmarshalSunny(t *testing.T) {
	rc := setupTestRedis(t)
	character.InitTemporalRegistry(rc)

	im := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("Atlas").
		SetLevel(1).
		SetExperience(0).
		Build()

	ctx := testTenantContext()
	res, err := model.Map(character.Transform(ctx))(model.FixedProvider(im))()
	if err != nil {
		t.Fatalf("Failed to transform model to rest model: %v", err)
	}

	rr := httptest.NewRecorder()
	server.MarshalResponse[character.RestModel](testLogger())(rr)(GetServer())(make(map[string][]string))(res)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	output := character.RestModel{}
	err = jsonapi.Unmarshal(body, &output)

	om, err := character.Extract(output)
	if err != nil {
		t.Fatalf("Failed to unmarshal rest model: %v", err)
	}
	if om.Id() != im.Id() {
		t.Fatalf("Failed to unmarshal rest model")
	}

	// do some basic tests
	if im.Id() != om.Id() {
		t.Fatalf("Input and output ids do not match")
	}
	if im.Name() != om.Name() {
		t.Fatalf("Input and output names do not match")
	}
}

func TestTransformExtractMapIdAndGmFields(t *testing.T) {
	rc := setupTestRedis(t)
	character.InitTemporalRegistry(rc)

	// Create a model with mapId and gm fields set
	im := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("TestCharacter").
		SetLevel(10).
		SetExperience(1000).
		SetMapId(_map.Id(100000001)). // Set a specific map ID
		SetGm(1).                     // Set GM status
		Build()

	// Test Transform method
	ctx := testTenantContext()
	restModel, err := character.Transform(ctx)(im)
	if err != nil {
		t.Fatalf("Failed to transform model to rest model: %v", err)
	}

	// Verify Transform method correctly maps mapId and gm fields
	if restModel.MapId != im.MapId() {
		t.Fatalf("Transform method failed to map mapId field correctly. Expected: %v, Got: %v", im.MapId(), restModel.MapId)
	}
	if restModel.Gm != im.GM() {
		t.Fatalf("Transform method failed to map gm field correctly. Expected: %v, Got: %v", im.GM(), restModel.Gm)
	}

	// Test Extract method
	extractedModel, err := character.Extract(restModel)
	if err != nil {
		t.Fatalf("Failed to extract model from rest model: %v", err)
	}

	// Verify Extract method correctly maps mapId and gm fields back to domain model
	if extractedModel.MapId() != im.MapId() {
		t.Fatalf("Extract method failed to map mapId field correctly. Expected: %v, Got: %v", im.MapId(), extractedModel.MapId())
	}
	if extractedModel.GM() != im.GM() {
		t.Fatalf("Extract method failed to map gm field correctly. Expected: %v, Got: %v", im.GM(), extractedModel.GM())
	}

	// Test with different values to ensure the fields are actually being mapped
	restModel2 := character.RestModel{
		Id:    2000,
		Name:  "TestCharacter2",
		MapId: _map.Id(100000002),
		Gm:    2,
	}

	extractedModel2, err := character.Extract(restModel2)
	if err != nil {
		t.Fatalf("Failed to extract model from rest model: %v", err)
	}

	if extractedModel2.MapId() != restModel2.MapId {
		t.Fatalf("Extract method failed to map different mapId. Expected: %v, Got: %v", restModel2.MapId, extractedModel2.MapId())
	}
	if extractedModel2.GM() != restModel2.Gm {
		t.Fatalf("Extract method failed to map different gm value. Expected: %v, Got: %v", restModel2.Gm, extractedModel2.GM())
	}
}
