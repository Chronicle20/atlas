package reactor

import (
	"github.com/Chronicle20/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
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

func TestRest(t *testing.T) {
	l, _ := test.NewNullLogger()

	input, err := Read(l)("", 0, fixedNodeProvider)()
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.MarshalResponse[RestModel](l)(rr)(GetServer())(map[string][]string{})(input)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	var output RestModel
	err = jsonapi.Unmarshal(body, &output)

	ok := compare(input, output)
	if !ok {
		t.Fatalf("Failed to compare model: %v", input.Id)
	}
}

func TestLinkedRest(t *testing.T) {
	l, _ := test.NewNullLogger()

	input, err := Read(l)("", 0, linkedNodeProvider)()
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.MarshalResponse[RestModel](l)(rr)(GetServer())(map[string][]string{})(input)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	var output RestModel
	err = jsonapi.Unmarshal(body, &output)

	ok := compare(input, output)
	if !ok {
		t.Fatalf("Failed to compare model: %v", input.Id)
	}
}

func compare(m1 RestModel, m2 RestModel) bool {
	return reflect.DeepEqual(m1, m2)
}
