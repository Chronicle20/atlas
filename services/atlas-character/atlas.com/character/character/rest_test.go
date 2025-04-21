package character_test

import (
	"atlas-character/character"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
)

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

func TestMarshalUnmarshalSunny(t *testing.T) {
	im := character.NewModelBuilder().
		SetAccountId(1000).
		SetWorldId(0).
		SetName("Atlas").
		SetLevel(1).
		SetExperience(0).
		Build()

	res, err := model.Map(character.Transform)(model.FixedProvider(im))()
	if err != nil {
		t.Fatalf("Failed to transform model to rest model: %v", err)
	}

	rr := httptest.NewRecorder()
	server.Marshal[character.RestModel](testLogger())(rr)(GetServer())(res)

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
