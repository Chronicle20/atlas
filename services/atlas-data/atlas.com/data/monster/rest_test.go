package monster

import (
	"atlas-data/xml"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"
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
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	_, _ = GetMonsterStringRegistry().Add(tt, MonsterString{id: strconv.Itoa(8510000), name: "Pianus"})
	_, _ = GetMonsterGaugeRegistry().Add(tt, Gauge{id: strconv.Itoa(8510000), exists: true})

	input, err := Read(l)(ctx)(xml.FromByteArrayProvider([]byte(testXML)))()
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
	if output.HpRecovery != 10000 || output.MpRecovery != 50000 {
		t.Fatalf("recovery fields lost in round-trip: got hp=%d mp=%d, want hp=10000 mp=50000",
			output.HpRecovery, output.MpRecovery)
	}
	if !ok {
		t.Fatalf("Failed to compare model: %v", input.Id)
	}
}

func compare(m1 RestModel, m2 RestModel) bool {
	return reflect.DeepEqual(m1, m2)
}

func TestRestModel_AttacksRoundTrip(t *testing.T) {
	in := RestModel{
		Id:   5100004,
		Name: "Samiho",
		Attacks: []AttackInfo{
			{Pos: 1, ConMP: 0, AttackAfter: 0},
			{Pos: 2, ConMP: 5, AttackAfter: 1500},
		},
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out RestModel
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(in.Attacks, out.Attacks) {
		t.Fatalf("Attacks round-trip mismatch:\n want %+v\n  got %+v", in.Attacks, out.Attacks)
	}
}
