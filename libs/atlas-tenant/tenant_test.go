package tenant

import (
	"encoding/json"
	"github.com/google/uuid"
	"testing"
)

func TestSerialization(t *testing.T) {
	id := uuid.New()
	region := "GMS"
	majorVersion := uint16(83)
	minorVersion := uint16(1)

	tenant, err := Create(id, region, majorVersion, minorVersion)
	if err != nil {
		t.Fatalf(err.Error())
	}

	data, err := json.Marshal(&tenant)
	if err != nil {
		t.Fatalf(err.Error())
	}

	var resTenant Model
	err = json.Unmarshal(data, &resTenant)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !tenant.Is(resTenant) {
		t.Fatalf("bad marshal / unmarshal")
	}
}
