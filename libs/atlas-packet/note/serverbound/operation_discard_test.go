package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v95 ida=0x624280
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v87 ida=0x684843
// packet-audit:verify packet=note/serverbound/NoteOperationDiscard version=gms_v83 ida=0x64aa57
func TestOperationDiscardRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationDiscard{
				count:          2,
				emptySlotCount: 3,
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
			if output.EmptySlotCount() != input.EmptySlotCount() {
				t.Errorf("emptySlotCount: got %v, want %v", output.EmptySlotCount(), input.EmptySlotCount())
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
