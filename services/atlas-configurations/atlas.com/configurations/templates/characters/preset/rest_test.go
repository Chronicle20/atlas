package preset

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAttributesCarriesAPAndSP(t *testing.T) {
	assertContains := func(t *testing.T, b []byte, want string) {
		t.Helper()
		if !strings.Contains(string(b), want) {
			t.Errorf("marshaled Attributes missing %q: %s", want, b)
		}
	}

	zero, err := json.Marshal(Attributes{})
	if err != nil {
		t.Fatalf("marshal zero: %v", err)
	}
	assertContains(t, zero, `"ap":0`)
	assertContains(t, zero, `"sp":""`)

	set, err := json.Marshal(Attributes{AP: 5, SP: "61,0,0"})
	if err != nil {
		t.Fatalf("marshal set: %v", err)
	}
	assertContains(t, set, `"ap":5`)
	assertContains(t, set, `"sp":"61,0,0"`)
}
