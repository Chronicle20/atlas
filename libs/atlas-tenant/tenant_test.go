package tenant

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestClientVariantDefaultsToModified(t *testing.T) {
	m, err := Create(uuid.New(), "GMS", 95, 1)
	if err != nil {
		t.Fatal(err)
	}
	if m.ClientVariant() != "modified" {
		t.Errorf("default: got %q, want modified", m.ClientVariant())
	}
}

func TestCreateWithVariant(t *testing.T) {
	m, err := CreateWithVariant(uuid.New(), "GMS", 95, 1, "stock")
	if err != nil {
		t.Fatal(err)
	}
	if m.ClientVariant() != "stock" {
		t.Errorf("got %q", m.ClientVariant())
	}
}

func TestJSONRoundTripVariant(t *testing.T) {
	m, _ := CreateWithVariant(uuid.New(), "GMS", 95, 1, "stock")
	js, err := json.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}
	var got Model
	if err := json.Unmarshal(js, &got); err != nil {
		t.Fatal(err)
	}
	if got.ClientVariant() != "stock" {
		t.Errorf("after roundtrip: %q", got.ClientVariant())
	}
}

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
