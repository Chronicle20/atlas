package quest

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

func TestRestModel(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Build a complete quest model through the readers
	quests := ReadQuestInfo(l)(xml.FromByteArrayProvider([]byte(testQuestInfoXML)))
	quests = ReadQuestCheck(l)(xml.FromByteArrayProvider([]byte(testCheckXML)))(quests)
	quests = ReadQuestAct(l)(xml.FromByteArrayProvider([]byte(testActXML)))(quests)

	input := quests[2000]

	rr := httptest.NewRecorder()
	server.MarshalResponse[RestModel](l)(rr)(GetServer())(map[string][]string{})(input)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model, status: %d", rr.Code)
	}

	body := rr.Body.Bytes()

	var output RestModel
	err := jsonapi.Unmarshal(body, &output)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	ok := compare(input, output)
	if !ok {
		t.Fatalf("Failed to compare model: input=%+v, output=%+v", input, output)
	}
}

func TestRestModelWithSkills(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Build a complete quest model through the readers
	quests := ReadQuestInfo(l)(xml.FromByteArrayProvider([]byte(testQuestInfoXML)))
	quests = ReadQuestCheck(l)(xml.FromByteArrayProvider([]byte(testCheckXML)))(quests)
	quests = ReadQuestAct(l)(xml.FromByteArrayProvider([]byte(testActXML)))(quests)

	input := quests[10000]

	rr := httptest.NewRecorder()
	server.MarshalResponse[RestModel](l)(rr)(GetServer())(map[string][]string{})(input)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model, status: %d", rr.Code)
	}

	body := rr.Body.Bytes()

	var output RestModel
	err := jsonapi.Unmarshal(body, &output)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	ok := compare(input, output)
	if !ok {
		t.Fatalf("Failed to compare model: input=%+v, output=%+v", input, output)
	}
}

func TestRestModelList(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Build complete quest models through the readers
	quests := ReadQuestInfo(l)(xml.FromByteArrayProvider([]byte(testQuestInfoXML)))
	quests = ReadQuestCheck(l)(xml.FromByteArrayProvider([]byte(testCheckXML)))(quests)
	quests = ReadQuestAct(l)(xml.FromByteArrayProvider([]byte(testActXML)))(quests)

	// Convert map to slice
	var inputList []RestModel
	for _, q := range quests {
		inputList = append(inputList, q)
	}

	rr := httptest.NewRecorder()
	server.MarshalResponse[[]RestModel](l)(rr)(GetServer())(map[string][]string{})(inputList)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model list, status: %d", rr.Code)
	}

	body := rr.Body.Bytes()

	var outputList []RestModel
	err := jsonapi.Unmarshal(body, &outputList)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(outputList) != len(inputList) {
		t.Fatalf("Expected %d quests, got %d", len(inputList), len(outputList))
	}
}

func TestRestModelGetID(t *testing.T) {
	m := RestModel{Id: 2000}
	if m.GetID() != "2000" {
		t.Fatalf("Expected GetID() to return '2000', got '%s'", m.GetID())
	}
}

func TestRestModelSetID(t *testing.T) {
	var m RestModel
	err := m.SetID("12345")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if m.Id != 12345 {
		t.Fatalf("Expected Id to be 12345, got %d", m.Id)
	}
}

func TestRestModelSetIDInvalid(t *testing.T) {
	var m RestModel
	err := m.SetID("invalid")
	if err == nil {
		t.Fatal("Expected SetID to fail with invalid input")
	}
}

func TestRestModelGetName(t *testing.T) {
	m := RestModel{}
	if m.GetName() != "quests" {
		t.Fatalf("Expected GetName() to return 'quests', got '%s'", m.GetName())
	}
}

func compare(m1 RestModel, m2 RestModel) bool {
	return reflect.DeepEqual(m1, m2)
}
