package serverbound

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// TestItemUseNoteDecodeTrailing pins the GMS v83/v84 arm shape: toName,
// message, trailing updateTime (task-137 design §1.1).
func TestItemUseNoteDecodeTrailing(t *testing.T) {
	raw := []byte{
		0x03, 0x00, 'B', 'o', 'b', // toName = "Bob"
		0x02, 0x00, 'h', 'i', // message = "hi"
		0x64, 0x00, 0x00, 0x00, // updateTime = 100 (trailing)
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := NewItemUseNote(false)
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.ToName() != "Bob" {
		t.Errorf("toName: got %q, want %q", p.ToName(), "Bob")
	}
	if p.Message() != "hi" {
		t.Errorf("message: got %q, want %q", p.Message(), "hi")
	}
	if p.UpdateTime() != 100 {
		t.Errorf("updateTime: got %d, want 100", p.UpdateTime())
	}
}

// TestItemUseNoteDecodeLeading pins the GMS v87/v95 and JMS185 arm shape:
// updateTime was already consumed by the leading ItemUse prefix, so the arm
// body is just toName, message.
func TestItemUseNoteDecodeLeading(t *testing.T) {
	raw := []byte{
		0x03, 0x00, 'B', 'o', 'b',
		0x02, 0x00, 'h', 'i',
	}
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	p := NewItemUseNote(true)
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})

	if p.ToName() != "Bob" {
		t.Errorf("toName: got %q, want %q", p.ToName(), "Bob")
	}
	if p.Message() != "hi" {
		t.Errorf("message: got %q, want %q", p.Message(), "hi")
	}
	if p.UpdateTime() != 0 {
		t.Errorf("updateTime: got %d, want 0 (leading variant reads none)", p.UpdateTime())
	}
}

func TestItemUseNoteRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			updateTimeFirst := (v.Region == "GMS" && v.MajorVersion >= 87) || v.Region == "JMS"
			input := ItemUseNote{toName: "Alice", message: "hello there", updateTime: 42, updateTimeFirst: updateTimeFirst}
			output := NewItemUseNote(updateTimeFirst)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ToName() != input.ToName() {
				t.Errorf("toName: got %q, want %q", output.ToName(), input.ToName())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %q, want %q", output.Message(), input.Message())
			}
			if !updateTimeFirst && output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %d, want %d", output.UpdateTime(), input.UpdateTime())
			}
		})
	}
}
