package account

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestExtractBirthDate confirms atlas-channel's hand-mirrored RestModel
// decodes the `birthDate` field atlas-account's RestModel puts on the wire.
// The two services' REST models are hand-mirrored, not generated, and drift
// silently (see PurchaseEventBody, a live instance of that drift) — this
// pins the birthDate field's wire tag agreement.
func TestExtractBirthDate(t *testing.T) {
	// The wire form atlas-account's RestModel produces for this field: a bare
	// uint32, json-tagged "birthDate".
	raw := []byte(`{"id":"456","name":"testuser","pin":"1234","pic":"5678","birthDate":19940203,"loggedIn":1}`)

	var rm RestModel
	if err := json.Unmarshal(raw, &rm); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if rm.BirthDate != 19940203 {
		t.Fatalf("RestModel.BirthDate mismatch after unmarshal. Expected 19940203, got %v", rm.BirthDate)
	}

	m, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}
	if m.BirthDate() != 19940203 {
		t.Errorf("Model.BirthDate mismatch after Extract. Expected 19940203, got %v", m.BirthDate())
	}
}

// TestTransformRoundTrip confirms Transform is the faithful inverse of
// Extract: every field Extract reads from RestModel survives a
// Transform -> Extract round trip.
func TestTransformRoundTrip(t *testing.T) {
	m := NewBuilder().
		SetId(456).
		SetName("testuser").
		SetPassword("pw123").
		SetPin("1234").
		SetPic("5678").
		SetBirthDate(19940203).
		SetLoggedIn(1).
		SetLastLogin(999).
		SetGender(2).
		SetBanned(true).
		SetTos(true).
		SetLanguage("en").
		SetCountry("us").
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	m2, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(m, m2) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, m2)
	}
}
