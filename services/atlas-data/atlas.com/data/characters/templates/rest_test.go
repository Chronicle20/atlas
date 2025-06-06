package templates

import (
	"atlas-data/xml"
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

	provider := xml.FromByteArrayProvider([]byte(testXML))
	rms := Read(l)(provider)
	res, err := rms()
	if err != nil {
		t.Fatal(err)
	}

	// Test marshaling and unmarshaling
	rr := httptest.NewRecorder()
	server.MarshalResponse[[]RestModel](l)(rr)(GetServer())(map[string][]string{})(res)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	var output []RestModel
	err = jsonapi.Unmarshal(body, &output)
	if err != nil {
		t.Fatal(err)
	}

	// Compare original and unmarshaled models
	if len(output) != len(res) {
		t.Fatalf("len(output) = %d, want %d", len(output), len(res))
	}

	for i := range res {
		ok := compare(res[i], output[i])
		if !ok {
			t.Fatalf("Failed to compare model at index %d", i)
		}
	}
}

func TestRestModelMethods(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Parse the XML to get a real model
	provider := xml.FromByteArrayProvider([]byte(testXML))
	rms := Read(l)(provider)
	res, err := rms()
	if err != nil {
		t.Fatal(err)
	}

	// Get the first model from the result
	if len(res) == 0 {
		t.Fatal("No models returned from XML parsing")
	}
	model := res[0]

	// Set a known ID for testing
	model.Id = 0

	if model.GetName() != "characterTemplates" {
		t.Fatalf("model.GetName() = %s, want characterTemplates", model.GetName())
	}

	// Test GetID
	if model.GetID() != "0" {
		t.Fatalf("model.GetID() = %s, want 0", model.GetID())
	}

	// Test SetID
	err = model.SetID("1")
	if err != nil {
		t.Fatalf("model.SetID() returned error: %v", err)
	}
	if model.Id != 1 {
		t.Fatalf("model.Id = %d, want 1", model.Id)
	}
}

func compare(m1 RestModel, m2 RestModel) bool {
	// Compare all fields except Id, which is generated randomly
	if m1.CharacterType != m2.CharacterType {
		return false
	}
	if !reflect.DeepEqual(m1.Faces, m2.Faces) {
		return false
	}
	if !reflect.DeepEqual(m1.HairStyles, m2.HairStyles) {
		return false
	}
	if !reflect.DeepEqual(m1.HairColors, m2.HairColors) {
		return false
	}
	if !reflect.DeepEqual(m1.SkinColors, m2.SkinColors) {
		return false
	}
	if !reflect.DeepEqual(m1.Tops, m2.Tops) {
		return false
	}
	if !reflect.DeepEqual(m1.Bottoms, m2.Bottoms) {
		return false
	}
	if !reflect.DeepEqual(m1.Shoes, m2.Shoes) {
		return false
	}
	if !reflect.DeepEqual(m1.Weapons, m2.Weapons) {
		return false
	}
	return true
}
