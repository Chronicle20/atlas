package note

import (
	"testing"
	"time"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestNoteDisplayRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			notes := []NoteEntry{
				{Id: 1, SenderName: "Alice", Message: "Hello!", Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC), Flag: 1},
				{Id: 2, SenderName: "Bob", Message: "Hi there", Timestamp: time.Date(2026, 2, 20, 14, 0, 0, 0, time.UTC), Flag: 0},
			}
			input := NewNoteDisplay(3, notes)
			output := Display{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if len(output.Notes()) != 2 {
				t.Fatalf("notes: got %d, want 2", len(output.Notes()))
			}
			if output.Notes()[0].Id != 1 {
				t.Errorf("note[0].Id: got %v, want 1", output.Notes()[0].Id)
			}
			if output.Notes()[0].SenderName != "Alice" {
				t.Errorf("note[0].SenderName: got %q, want %q", output.Notes()[0].SenderName, "Alice")
			}
			if output.Notes()[0].Message != "Hello!" {
				t.Errorf("note[0].Message: got %q, want %q", output.Notes()[0].Message, "Hello!")
			}
			if output.Notes()[0].Flag != 1 {
				t.Errorf("note[0].Flag: got %v, want 1", output.Notes()[0].Flag)
			}
			if output.Notes()[1].SenderName != "Bob" {
				t.Errorf("note[1].SenderName: got %q, want %q", output.Notes()[1].SenderName, "Bob")
			}
		})
	}
}

func TestNoteDisplayEmptyRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewNoteDisplay(3, nil)
	output := Display{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if len(output.Notes()) != 0 {
		t.Errorf("notes: got %d, want 0", len(output.Notes()))
	}
}
