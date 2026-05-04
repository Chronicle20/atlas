package character_test

import (
	"atlas-character/character"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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
	res, err := model.Map(character.Transform(testLogger(), ctx))(model.FixedProvider(im))()
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

// task-055: MapId/Instance are no longer model-owned; Transform pulls them
// in-flight from atlas-maps. The dedicated Transform/Extract MapId test was
// removed with the model fields. GM round-trip is exercised below.
func TestTransformExtractGmField(t *testing.T) {
	rc := setupTestRedis(t)
	character.InitTemporalRegistry(rc)

	im := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("TestCharacter").
		SetLevel(10).
		SetExperience(1000).
		SetGm(1).
		Build()

	ctx := testTenantContext()
	// Transform issues an atlas-maps lookup; in unit tests no such service
	// exists, so it falls back to zero values for MapId/Instance — that's
	// the expected D11 behavior.
	restModel, err := character.Transform(testLogger(), ctx)(im)
	if err != nil {
		t.Fatalf("Failed to transform model to rest model: %v", err)
	}

	if restModel.Gm != im.GM() {
		t.Fatalf("Transform method failed to map gm field correctly. Expected: %v, Got: %v", im.GM(), restModel.Gm)
	}

	extractedModel, err := character.Extract(restModel)
	if err != nil {
		t.Fatalf("Failed to extract model from rest model: %v", err)
	}

	if extractedModel.GM() != im.GM() {
		t.Fatalf("Extract method failed to map gm field correctly. Expected: %v, Got: %v", im.GM(), extractedModel.GM())
	}
}
