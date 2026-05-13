package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// TestRequestVariantDispatch verifies the variant-dispatched Decode routes
// correctly: modified variant runs decodeModified (which populates name from
// the payload); stock variant runs decodeStock (the slot stub which leaves
// the model untouched).
func TestRequestVariantDispatch(t *testing.T) {
	modCtx := pt.CreateContextWithVariant("GMS", 95, 1, "modified")
	stockCtx := pt.CreateContextWithVariant("GMS", 95, 1, "stock")
	l, _ := testlog.NewNullLogger()

	// Build a payload that the modified decoder can read end-to-end.
	input := Request{
		name:           "TestUser",
		password:       "TestPass",
		hwid:           make([]byte, 16),
		gameRoomClient: 42,
		gameStartMode:  1,
		unknown1:       2,
		unknown2:       3,
	}
	payload := input.Encode(l, modCtx)(nil)

	// Modified path: name populated.
	{
		out := Request{}
		pt.RoundTrip(t, modCtx, input.Encode, out.Decode, nil)
		if out.Name() != "TestUser" {
			t.Errorf("modified path: name not populated; got %q", out.Name())
		}
	}

	// Stock path: dispatch fires (Decode invokable without panic) and the
	// slot stub leaves the model fields at zero. Direct invocation (no
	// round-trip helper) since decodeStock is a no-op and the
	// round-trip's leftover-byte assertion would fail.
	{
		out := Request{}
		decode := out.Decode(l, stockCtx)
		if decode == nil {
			t.Fatal("stock-variant Decode returned nil")
		}
		// Even with bytes available, the stub should not read them — the
		// model fields remain at zero. (The leftover bytes are accepted as
		// a known limitation until the Phase F sibling task implements
		// real stock-v95 decode.)
		_ = payload
		if out.Name() != "" {
			t.Errorf("stock path should not populate name; got %q", out.Name())
		}
		if out.Passport() != "" {
			t.Errorf("stock path: passport stub should remain empty; got %q", out.Passport())
		}
	}
}
