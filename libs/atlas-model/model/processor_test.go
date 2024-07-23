package model

import "testing"

func TestFirstNoFilter(t *testing.T) {
	p := FixedProvider([]uint32{1})
	r, err := First(p)
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if r != 1 {
		t.Errorf("Expected 1, got %d", r)
	}
}

func byTwo(val uint32) (uint32, error) {
	return val * 2, nil
}

func TestMap(t *testing.T) {
	p := FixedProvider(uint32(1))
	mp := Map[uint32, uint32](p, byTwo)

	ar, err := mp()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	er, _ := p()
	if er*2 != ar {
		t.Errorf("Expected %d, got %d", er*2, ar)
	}
}

func TestSliceMap(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	mp := SliceMap[uint32, uint32](p, byTwo)

	ar, err := mp()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	er, _ := p()
	for i := range er {
		if er[i]*2 != ar[i] {
			t.Errorf("Expected %d, got %d", er[i]*2, ar[i])
		}
	}
}

func TestParallelSliceMap(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	mp := SliceMap[uint32, uint32](p, byTwo, ParallelMap())

	ar, err := mp()
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}

	er, _ := p()
	for i := range er {
		if er[i]*2 != ar[i] {
			t.Errorf("Expected %d, got %d", er[i]*2, ar[i])
		}
	}
}

func isTwo(val uint32) bool {
	return val == 2
}

func TestFirst(t *testing.T) {
	p := FixedProvider([]uint32{1, 2, 3, 4, 5})
	mp, err := First(p, isTwo)
	if err != nil {
		t.Errorf("Expected result, got err %s", err)
	}
	if mp != 2 {
		t.Errorf("Expected 2, got %d", mp)
	}
}
