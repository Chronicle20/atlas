package character

import (
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// TestTransformRoundTrip asserts TransformForeign and ExtractForeign are
// exact inverses of each other over the mapped fields of ForeignModel.
func TestTransformRoundTrip(t *testing.T) {
	t.Run("foreign", func(t *testing.T) {
		m := ForeignModel{
			id:      42,
			worldId: world.Id(1),
			name:    "Bob",
			level:   99,
			jobId:   job.Id(200),
			gm:      2,
		}

		rm, err := TransformForeign(m)
		if err != nil {
			t.Fatalf("TransformForeign: %v", err)
		}
		got, err := ExtractForeign(rm)
		if err != nil {
			t.Fatalf("ExtractForeign: %v", err)
		}
		if !reflect.DeepEqual(got, m) {
			t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
		}
	})
}
