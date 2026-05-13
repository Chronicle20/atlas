package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func TestRequestStockVariantDispatch(t *testing.T) {
	ctx := pt.CreateContextWithVariant("GMS", 95, 1, "stock")
	r := Request{}
	l, _ := testlog.NewNullLogger()
	dec := r.Decode(l, ctx)
	if dec == nil {
		t.Fatal("nil decoder")
	}
	// The dispatch path exists; the Passport accessor returns the zero value
	// ("") when the field was never set by a real stock-v95 payload.
	if r.Passport() != "" {
		t.Errorf("passport should default empty; got %q", r.Passport())
	}
}
