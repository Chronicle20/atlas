package note

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationDiscardRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationDiscard{
				count: 2,
				val1:  1,
				val2:  0,
				entries: []DiscardEntry{
					{id: 100, flag: 1},
					{id: 200, flag: 2},
				},
			}
			output := OperationDiscard{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Count() != input.Count() {
				t.Errorf("count: got %v, want %v", output.Count(), input.Count())
			}
			if output.Val1() != input.Val1() {
				t.Errorf("val1: got %v, want %v", output.Val1(), input.Val1())
			}
			if output.Val2() != input.Val2() {
				t.Errorf("val2: got %v, want %v", output.Val2(), input.Val2())
			}
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries length: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i, e := range output.Entries() {
				if e.Id() != input.Entries()[i].Id() {
					t.Errorf("entry[%d].id: got %v, want %v", i, e.Id(), input.Entries()[i].Id())
				}
				if e.Flag() != input.Entries()[i].Flag() {
					t.Errorf("entry[%d].flag: got %v, want %v", i, e.Flag(), input.Entries()[i].Flag())
				}
			}
		})
	}
}
