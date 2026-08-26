package wzxml

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

func TestPropertyToElement(t *testing.T) {
	tests := []struct {
		name      string
		input     property.Property
		wantLocal string
		check     func(t *testing.T, e Element)
	}{
		{
			name:      "null",
			input:     property.NewNull("a"),
			wantLocal: "null",
			check: func(t *testing.T, e Element) {
				if e.Name != "a" {
					t.Errorf("Name = %q, want %q", e.Name, "a")
				}
			},
		},
		{
			name:      "short",
			input:     property.NewShort("a", -3),
			wantLocal: "short",
			check: func(t *testing.T, e Element) {
				if e.Value != "-3" {
					t.Errorf("Value = %q, want %q", e.Value, "-3")
				}
			},
		},
		{
			name:      "int",
			input:     property.NewInt("state", 1),
			wantLocal: "int",
			check: func(t *testing.T, e Element) {
				if e.Value != "1" {
					t.Errorf("Value = %q, want %q", e.Value, "1")
				}
			},
		},
		{
			name:      "long",
			input:     property.NewLong("a", 9000000000),
			wantLocal: "long",
			check: func(t *testing.T, e Element) {
				if e.Value != "9000000000" {
					t.Errorf("Value = %q, want %q", e.Value, "9000000000")
				}
			},
		},
		{
			name:      "float",
			input:     property.NewFloat("a", 1.5),
			wantLocal: "float",
			check: func(t *testing.T, e Element) {
				if e.Value != "1.5" {
					t.Errorf("Value = %q, want %q", e.Value, "1.5")
				}
			},
		},
		{
			name:      "float integral",
			input:     property.NewFloat("a", 2),
			wantLocal: "float",
			check: func(t *testing.T, e Element) {
				if e.Value != "2.0" {
					t.Errorf("Value = %q, want %q", e.Value, "2.0")
				}
			},
		},
		{
			name:      "double",
			input:     property.NewDouble("a", 0),
			wantLocal: "double",
			check: func(t *testing.T, e Element) {
				if e.Value != "0.0" {
					t.Errorf("Value = %q, want %q", e.Value, "0.0")
				}
			},
		},
		{
			name:      "string",
			input:     property.NewString("name", "Red Potion"),
			wantLocal: "string",
			check: func(t *testing.T, e Element) {
				if e.Value != "Red Potion" {
					t.Errorf("Value = %q, want %q", e.Value, "Red Potion")
				}
			},
		},
		{
			name: "sub",
			input: property.NewSub("event", []property.Property{
				property.NewInt("state", 1),
			}),
			wantLocal: "imgdir",
			check: func(t *testing.T, e Element) {
				if e.Name != "event" {
					t.Errorf("Name = %q, want %q", e.Name, "event")
				}
				if len(e.Children) != 1 {
					t.Fatalf("len(Children) = %d, want 1", len(e.Children))
				}
				if e.Children[0].XMLName.Local != "int" || e.Children[0].Value != "1" {
					t.Errorf("Children[0] = %+v", e.Children[0])
				}
			},
		},
		{
			name: "canvas",
			input: property.NewCanvas("0", 100, 121, 2, 0x1000, 64, []property.Property{
				property.NewVector("origin", 49, 121),
			}),
			wantLocal: "canvas",
			check: func(t *testing.T, e Element) {
				if e.Width != "100" {
					t.Errorf("Width = %q, want %q", e.Width, "100")
				}
				if e.Height != "121" {
					t.Errorf("Height = %q, want %q", e.Height, "121")
				}
				if len(e.Children) != 1 {
					t.Fatalf("len(Children) = %d, want 1", len(e.Children))
				}
				if e.Children[0].XMLName.Local != "vector" || e.Children[0].X != "49" || e.Children[0].Y != "121" {
					t.Errorf("Children[0] = %+v", e.Children[0])
				}
			},
		},
		{
			name:      "vector",
			input:     property.NewVector("lt", -100, -100),
			wantLocal: "vector",
			check: func(t *testing.T, e Element) {
				if e.X != "-100" || e.Y != "-100" {
					t.Errorf("X=%q Y=%q, want -100/-100", e.X, e.Y)
				}
			},
		},
		{
			name: "convex",
			input: property.NewConvex("a", []property.Property{
				property.NewVector("0", 1, 2),
			}),
			wantLocal: "extended",
			check: func(t *testing.T, e Element) {
				if len(e.Children) != 1 {
					t.Fatalf("len(Children) = %d, want 1", len(e.Children))
				}
			},
		},
		{
			name:      "sound",
			input:     property.NewSound("a"),
			wantLocal: "sound",
			check: func(t *testing.T, e Element) {
				if e.Name != "a" {
					t.Errorf("Name = %q, want %q", e.Name, "a")
				}
				if e.Value != "" {
					t.Errorf("Value = %q, want empty", e.Value)
				}
			},
		},
		{
			name:      "uol",
			input:     property.NewUOL("0", "../0/0"),
			wantLocal: "uol",
			check: func(t *testing.T, e Element) {
				if e.Value != "../0/0" {
					t.Errorf("Value = %q, want %q", e.Value, "../0/0")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := PropertyToElement(tc.input)
			if e.XMLName.Local != tc.wantLocal {
				t.Errorf("XMLName.Local = %q, want %q", e.XMLName.Local, tc.wantLocal)
			}
			tc.check(t, e)
		})
	}
}

func TestFormatFloat(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0.0"},
		{1.5, "1.5"},
		{-2, "-2.0"},
		{100, "100.0"},
	}
	for _, tc := range tests {
		if got := FormatFloat(tc.in); got != tc.want {
			t.Errorf("FormatFloat(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPropertiesToElements(t *testing.T) {
	if got := PropertiesToElements(nil); got != nil {
		t.Errorf("PropertiesToElements(nil) = %+v, want nil", got)
	}

	props := []property.Property{
		property.NewInt("a", 1),
		property.NewInt("b", 2),
	}
	got := PropertiesToElements(props)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "a" || got[1].Name != "b" {
		t.Errorf("order not preserved: %+v", got)
	}
}
