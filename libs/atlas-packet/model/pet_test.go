package model

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestPetRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Pet{
				TemplateId:  5000001,
				Name:        "Kitty",
				Id:          12345,
				X:           100,
				Y:           200,
				Stance:      2,
				Foothold:    50,
				NameTag:     1,
				ChatBalloon: 1,
			}
			output := Pet{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.TemplateId != input.TemplateId {
				t.Errorf("templateId: got %v, want %v", output.TemplateId, input.TemplateId)
			}
			if output.Name != input.Name {
				t.Errorf("name: got %v, want %v", output.Name, input.Name)
			}
			if output.Id != input.Id {
				t.Errorf("id: got %v, want %v", output.Id, input.Id)
			}
			if output.X != input.X {
				t.Errorf("x: got %v, want %v", output.X, input.X)
			}
			if output.Y != input.Y {
				t.Errorf("y: got %v, want %v", output.Y, input.Y)
			}
			if output.Stance != input.Stance {
				t.Errorf("stance: got %v, want %v", output.Stance, input.Stance)
			}
			if output.Foothold != input.Foothold {
				t.Errorf("foothold: got %v, want %v", output.Foothold, input.Foothold)
			}
		})
	}
}
