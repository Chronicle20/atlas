package _map

import (
	"atlas-data/map/object"
	npc2 "atlas-data/npc"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	GetMapStringRegistry().Add(tt, MapString{
		id:         strconv.Itoa(50000),
		mapName:    "Dangerous Forest",
		streetName: "Maple Road",
	})
	npc2.GetNpcStringRegistry().Add(tt, npc2.NewNpcString(strconv.Itoa(2003), "Robin"))
	npc2.GetNpcStringRegistry().Add(tt, npc2.NewNpcString(strconv.Itoa(2005), "Sam"))

	input, err := Read(l)(ctx)("", 0, fixedNodeProvider)()
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
	tt := testTenant()
	l, _ := test.NewNullLogger()
	ctx := tenant.WithContext(context.Background(), tt)

	GetMapStringRegistry().Add(tt, MapString{
		id:         strconv.Itoa(100020100),
		mapName:    "Henesys Pig Farm",
		streetName: "Mini Dungeon",
	})

	input, err := Read(l)(ctx)("", 0, linkedNodeProvider)()
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

func TestMapObjectsSurviveJSONAPIRoundTrip(t *testing.T) {
	m := RestModel{
		Id: 50000,
		Objects: []object.RestModel{
			{
				Id:           "ENVIRONMENT:gate",
				Kind:         "ENVIRONMENT",
				Name:         "gate",
				ObjectSource: "effect",
				L0:           "quest",
				L1:           "gate",
				L2:           "1",
				X:            640,
				Y:            120,
				Z:            0,
				Layer:        3,
			},
			{
				Id:           "OBSTACLE:menhir0",
				Kind:         "OBSTACLE",
				Name:         "menhir0",
				ObjectSource: "trapGL",
				L0:           "ckPQ",
				L1:           "menhir",
				L2:           "0",
				X:            -30,
				Y:            45,
				Z:            7,
				Layer:        2,
			},
		},
	}

	d, err := jsonapi.MarshalToStruct(m, nil)
	if err != nil {
		t.Fatalf("failed to marshal to struct: %v", err)
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var included []map[string]interface{}
	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("failed to unmarshal document for inspection: %v", err)
	}
	rawIncluded, ok := doc["included"].([]interface{})
	if !ok {
		t.Fatalf("expected included array in document, got %v", doc["included"])
	}
	for _, ri := range rawIncluded {
		if m, ok := ri.(map[string]interface{}); ok {
			included = append(included, m)
		}
	}
	objectCount := 0
	foundIds := make(map[string]bool)
	for _, inc := range included {
		if inc["type"] == "map-objects" {
			objectCount++
			if id, ok := inc["id"].(string); ok {
				foundIds[id] = true
			}
		}
	}
	if objectCount != 2 {
		t.Fatalf("expected 2 map-objects entries in included, got %d", objectCount)
	}
	if !foundIds["ENVIRONMENT:gate"] || !foundIds["OBSTACLE:menhir0"] {
		t.Fatalf("expected included ids ENVIRONMENT:gate and OBSTACLE:menhir0, got %v", foundIds)
	}

	var out RestModel
	if err := jsonapi.Unmarshal(data, &out); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(out.Objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(out.Objects))
	}

	sort.Slice(out.Objects, func(i, j int) bool {
		return out.Objects[i].Id < out.Objects[j].Id
	})

	expected := []object.RestModel{m.Objects[0], m.Objects[1]}
	sort.Slice(expected, func(i, j int) bool {
		return expected[i].Id < expected[j].Id
	})

	for i := range expected {
		if out.Objects[i].Kind != expected[i].Kind {
			t.Errorf("object %d Kind: expected %q, got %q", i, expected[i].Kind, out.Objects[i].Kind)
		}
		if out.Objects[i].Name != expected[i].Name {
			t.Errorf("object %d Name: expected %q, got %q", i, expected[i].Name, out.Objects[i].Name)
		}
		if out.Objects[i].ObjectSource != expected[i].ObjectSource {
			t.Errorf("object %d ObjectSource: expected %q, got %q", i, expected[i].ObjectSource, out.Objects[i].ObjectSource)
		}
		if out.Objects[i].L0 != expected[i].L0 {
			t.Errorf("object %d L0: expected %q, got %q", i, expected[i].L0, out.Objects[i].L0)
		}
		if out.Objects[i].L1 != expected[i].L1 {
			t.Errorf("object %d L1: expected %q, got %q", i, expected[i].L1, out.Objects[i].L1)
		}
		if out.Objects[i].L2 != expected[i].L2 {
			t.Errorf("object %d L2: expected %q, got %q", i, expected[i].L2, out.Objects[i].L2)
		}
		if out.Objects[i].X != expected[i].X {
			t.Errorf("object %d X: expected %d, got %d", i, expected[i].X, out.Objects[i].X)
		}
		if out.Objects[i].Y != expected[i].Y {
			t.Errorf("object %d Y: expected %d, got %d", i, expected[i].Y, out.Objects[i].Y)
		}
		if out.Objects[i].Z != expected[i].Z {
			t.Errorf("object %d Z: expected %d, got %d", i, expected[i].Z, out.Objects[i].Z)
		}
		if out.Objects[i].Layer != expected[i].Layer {
			t.Errorf("object %d Layer: expected %d, got %d", i, expected[i].Layer, out.Objects[i].Layer)
		}
	}
}
