package object

import "testing"

func TestTransformRoundTripsEveryField(t *testing.T) {
	m := NewBuilder().
		SetKind("ENVIRONMENT").
		SetName("gate").
		SetObjectSource("effect").
		SetL0("quest").
		SetL1("gate").
		SetL2("1").
		SetX(640).
		SetY(120).
		SetZ(5).
		SetLayer(1).
		Build()

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform returned error: %v", err)
	}

	if rm.Id != "ENVIRONMENT:gate" {
		t.Errorf("Id = %q, want %q", rm.Id, "ENVIRONMENT:gate")
	}
	if rm.Kind != "ENVIRONMENT" {
		t.Errorf("Kind = %q, want %q", rm.Kind, "ENVIRONMENT")
	}
	if rm.Name != "gate" {
		t.Errorf("Name = %q, want %q", rm.Name, "gate")
	}
	if rm.ObjectSource != "effect" {
		t.Errorf("ObjectSource = %q, want %q", rm.ObjectSource, "effect")
	}
	if rm.L0 != "quest" {
		t.Errorf("L0 = %q, want %q", rm.L0, "quest")
	}
	if rm.L1 != "gate" {
		t.Errorf("L1 = %q, want %q", rm.L1, "gate")
	}
	if rm.L2 != "1" {
		t.Errorf("L2 = %q, want %q", rm.L2, "1")
	}
	if rm.X != 640 {
		t.Errorf("X = %d, want %d", rm.X, 640)
	}
	if rm.Y != 120 {
		t.Errorf("Y = %d, want %d", rm.Y, 120)
	}
	if rm.Z != 5 {
		t.Errorf("Z = %d, want %d", rm.Z, 5)
	}
	if rm.Layer != 1 {
		t.Errorf("Layer = %d, want %d", rm.Layer, 1)
	}
}

func TestTransformSlice(t *testing.T) {
	ms := []Model{
		NewBuilder().SetKind("ENVIRONMENT").SetName("gate").Build(),
		NewBuilder().SetKind("OBSTACLE").SetName("rock").Build(),
	}

	rms, err := TransformSlice(ms)
	if err != nil {
		t.Fatalf("TransformSlice returned error: %v", err)
	}
	if len(rms) != 2 {
		t.Fatalf("len(rms) = %d, want 2", len(rms))
	}
	if rms[0].Id != "ENVIRONMENT:gate" {
		t.Errorf("rms[0].Id = %q, want %q", rms[0].Id, "ENVIRONMENT:gate")
	}
	if rms[1].Id != "OBSTACLE:rock" {
		t.Errorf("rms[1].Id = %q, want %q", rms[1].Id, "OBSTACLE:rock")
	}
}

func TestTransformSliceEmpty(t *testing.T) {
	rms, err := TransformSlice(nil)
	if err != nil {
		t.Fatalf("TransformSlice returned error: %v", err)
	}
	if rms == nil {
		t.Fatal("TransformSlice(nil) returned nil slice, want empty slice")
	}
	if len(rms) != 0 {
		t.Fatalf("len(rms) = %d, want 0", len(rms))
	}
}
