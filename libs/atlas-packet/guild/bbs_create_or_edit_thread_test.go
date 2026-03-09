package guild

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestBBSCreateOrEditThreadRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/create", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSCreateOrEditThread{modify: false, notice: true, title: "Hello", message: "World", emoticonId: 5}
			output := BBSCreateOrEditThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Modify() != input.Modify() {
				t.Errorf("modify: got %v, want %v", output.Modify(), input.Modify())
			}
			if output.Notice() != input.Notice() {
				t.Errorf("notice: got %v, want %v", output.Notice(), input.Notice())
			}
			if output.Title() != input.Title() {
				t.Errorf("title: got %v, want %v", output.Title(), input.Title())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.EmoticonId() != input.EmoticonId() {
				t.Errorf("emoticonId: got %v, want %v", output.EmoticonId(), input.EmoticonId())
			}
		})
		t.Run(v.Name+"/edit", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSCreateOrEditThread{modify: true, threadId: 42, notice: false, title: "Updated", message: "Content", emoticonId: 3}
			output := BBSCreateOrEditThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Modify() != input.Modify() {
				t.Errorf("modify: got %v, want %v", output.Modify(), input.Modify())
			}
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
			if output.Notice() != input.Notice() {
				t.Errorf("notice: got %v, want %v", output.Notice(), input.Notice())
			}
			if output.Title() != input.Title() {
				t.Errorf("title: got %v, want %v", output.Title(), input.Title())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.EmoticonId() != input.EmoticonId() {
				t.Errorf("emoticonId: got %v, want %v", output.EmoticonId(), input.EmoticonId())
			}
		})
	}
}
