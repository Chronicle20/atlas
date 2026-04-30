package character

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateCharacterCommandBody_GmMeso_RoundTrip(t *testing.T) {
	in := CreateCharacterCommandBody{Gm: 2, Meso: 12345}
	bs, _ := json.Marshal(in)
	if !strings.Contains(string(bs), `"gm":2`) || !strings.Contains(string(bs), `"meso":12345`) {
		t.Fatalf("expected gm/meso in JSON, got %s", string(bs))
	}
	var out CreateCharacterCommandBody
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Gm != 2 || out.Meso != 12345 {
		t.Fatalf("expected gm=2 meso=12345, got gm=%d meso=%d", out.Gm, out.Meso)
	}
}
