package tenant

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSerialization(t *testing.T) {
	id := uuid.New()
	region := "GMS"
	majorVersion := uint16(83)
	minorVersion := uint16(1)

	tenant, err := Register(id, region, majorVersion, minorVersion)
	if err != nil {
		t.Fatal(err.Error())
	}

	data, err := json.Marshal(&tenant)
	if err != nil {
		t.Fatal(err.Error())
	}

	var resTenant Model
	err = json.Unmarshal(data, &resTenant)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !tenant.Is(resTenant) {
		t.Fatalf("bad marshal / unmarshal")
	}
}
