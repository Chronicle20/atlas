package model

import (
	"errors"
	"strconv"
	"testing"
)

func TestMapPagedPreservesEnvelope(t *testing.T) {
	src := FixedProvider(Paged[int]{Items: []int{1, 2, 3}, Total: 42, Page: Page{Number: 2, Size: 3}})
	out, err := MapPaged[int, string](func(i int) (string, error) { return strconv.Itoa(i), nil })(src)()()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 42 || out.Page.Number != 2 || out.Page.Size != 3 {
		t.Fatalf("envelope not preserved: %+v", out)
	}
	if len(out.Items) != 3 || out.Items[0] != "1" || out.Items[2] != "3" {
		t.Fatalf("items wrong: %v", out.Items)
	}
}

func TestMapPagedParallelIndexStable(t *testing.T) {
	items := make([]int, 200)
	for i := range items {
		items[i] = i
	}
	src := FixedProvider(Paged[int]{Items: items, Total: 200, Page: Page{Number: 1, Size: 200}})
	out, err := MapPaged[int, int](func(i int) (int, error) { return i * 2, nil })(src)(ParallelMap())()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range out.Items {
		if v != i*2 {
			t.Fatalf("index %d: got %d want %d", i, v, i*2)
		}
	}
}

func TestMapPagedSourceError(t *testing.T) {
	want := errors.New("boom")
	src := ErrorProvider[Paged[int]](want)
	_, err := MapPaged[int, int](func(i int) (int, error) { return i, nil })(src)()()
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}

func TestMapPagedTransformError(t *testing.T) {
	want := errors.New("boom")
	src := FixedProvider(Paged[int]{Items: []int{1}, Total: 1, Page: Page{Number: 1, Size: 1}})
	_, err := MapPaged[int, int](func(int) (int, error) { return 0, want })(src)()()
	if !errors.Is(err, want) {
		t.Fatalf("got %v want %v", err, want)
	}
}
