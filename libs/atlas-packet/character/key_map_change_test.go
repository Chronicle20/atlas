package character

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestKeyMapChangeMode0RoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := KeyMapChange{
				mode: 0,
				entries: []KeyMapEntry{
					{KeyId: 2, TheType: 6, Action: 100},
					{KeyId: 63, TheType: 6, Action: 200},
				},
			}
			output := KeyMapChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if len(output.Entries()) != len(input.Entries()) {
				t.Fatalf("entries count: got %v, want %v", len(output.Entries()), len(input.Entries()))
			}
			for i, e := range output.Entries() {
				if e.KeyId != input.entries[i].KeyId {
					t.Errorf("entries[%d].KeyId: got %v, want %v", i, e.KeyId, input.entries[i].KeyId)
				}
				if e.TheType != input.entries[i].TheType {
					t.Errorf("entries[%d].TheType: got %v, want %v", i, e.TheType, input.entries[i].TheType)
				}
				if e.Action != input.entries[i].Action {
					t.Errorf("entries[%d].Action: got %v, want %v", i, e.Action, input.entries[i].Action)
				}
			}
		})
	}
}

func TestKeyMapChangeMode1RoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := KeyMapChange{mode: 1, itemId: 2001000}
			output := KeyMapChange{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}
