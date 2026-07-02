package paginate

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func TestSlice(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7}
	p1 := Slice(items, model.Page{Number: 1, Size: 3})
	if p1.Total != 7 || len(p1.Items) != 3 || p1.Items[0] != 1 {
		t.Fatalf("page1: %+v", p1)
	}
	p3 := Slice(items, model.Page{Number: 3, Size: 3})
	if len(p3.Items) != 1 || p3.Items[0] != 7 {
		t.Fatalf("partial last page: %+v", p3)
	}
	past := Slice(items, model.Page{Number: 9, Size: 3})
	if len(past.Items) != 0 || past.Total != 7 {
		t.Fatalf("past-end: %+v", past)
	}
	empty := Slice([]int{}, model.Page{Number: 1, Size: 3})
	if len(empty.Items) != 0 || empty.Total != 0 {
		t.Fatalf("empty: %+v", empty)
	}
}

func TestEnvelopeFor(t *testing.T) {
	env := EnvelopeFor(model.Paged[int]{Items: []int{1}, Total: 9, Page: model.Page{Number: 2, Size: 4}})
	if env.Total != 9 || env.PageNumber != 2 || env.PageSize != 4 {
		t.Fatalf("%+v", env)
	}
}
