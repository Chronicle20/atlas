package tenant

import (
	"context"
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

func TestContextRoundTripPreservesVariant(t *testing.T) {
	orig, err := CreateWithVariant(uuid.New(), "GMS", 95, 1, "stock")
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithContext(context.Background(), orig)
	got := MustFromContext(ctx)
	if got.ClientVariant() != "stock" {
		t.Errorf("context round-trip lost variant: got %q, want stock", got.ClientVariant())
	}
}

func TestContextDefaultVariantWhenMissing(t *testing.T) {
	// A context populated by some older caller that doesn't set the variant key
	// should still produce a usable tenant whose ClientVariant() defaults to "modified".
	ctx := context.WithValue(context.Background(), ID, uuid.New())
	ctx = context.WithValue(ctx, Region, "GMS")
	ctx = context.WithValue(ctx, MajorVersion, uint16(83))
	ctx = context.WithValue(ctx, MinorVersion, uint16(1))
	got, err := FromContext(ctx)()
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientVariant() != "modified" {
		t.Errorf("missing variant key: got %q, want modified default", got.ClientVariant())
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
