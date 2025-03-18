package consumable

import (
	"atlas-data/xml"
	"github.com/Chronicle20/atlas-model/model"
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

	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	res, err := rms()
	if err != nil {
		t.Fatal(err)
	}
	irmm, err := model.CollectToMap[RestModel, uint32, RestModel](rms, RestModel.GetId, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	server.MarshalResponse[[]RestModel](l)(rr)(GetServer())(map[string][]string{})(res)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to write rest model: %v", err)
	}

	body := rr.Body.Bytes()

	var output []RestModel
	err = jsonapi.Unmarshal(body, &output)

	ormm, err := model.CollectToMap[RestModel, uint32, RestModel](model.FixedProvider(output), RestModel.GetId, Identity)()
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range irmm {
		ok := compare(v, ormm[k])
		if !ok {
			t.Fatalf("Failed to compare model: %v", k)
		}
	}
}

func compare(m1 RestModel, m2 RestModel) bool {
	return reflect.DeepEqual(m1, m2)
}
