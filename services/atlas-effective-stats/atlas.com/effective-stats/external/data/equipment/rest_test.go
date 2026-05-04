package equipment

import (
	"encoding/json"
	"testing"
)

func TestRestModel_DecodesAtlasDataFields(t *testing.T) {
	body := []byte(`{"reqLevel":40,"reqJob":2,"reqStr":0,"reqDex":0,"reqInt":80,"reqLuk":40}`)
	var rm RestModel
	if err := json.Unmarshal(body, &rm); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if rm.ReqLevel != 40 || rm.ReqJob != 2 || rm.ReqInt != 80 || rm.ReqLuk != 40 {
		t.Errorf("decode mismatch: %+v", rm)
	}
}

func TestRestModel_GetNameIsStatistics(t *testing.T) {
	if (RestModel{}).GetName() != "statistics" {
		t.Errorf("GetName mismatch")
	}
}

func TestRestModel_IDRoundTrip(t *testing.T) {
	var rm RestModel
	if err := rm.SetID("1052095"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if rm.GetID() != "1052095" {
		t.Errorf("GetID = %s", rm.GetID())
	}
}
