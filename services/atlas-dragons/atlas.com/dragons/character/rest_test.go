package character

import (
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

func TestTransformRoundTrip(t *testing.T) {
	m, err := NewBuilder(1).
		SetJobId(job.Id(100)).
		SetX(200).
		SetY(300).
		SetStance(4).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	got, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if !reflect.DeepEqual(got, m) {
		t.Errorf("round trip mismatch. Expected %+v, got %+v", m, got)
	}
}
