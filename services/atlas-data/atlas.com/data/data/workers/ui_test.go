package workers

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
)

// TestFindItemProtectorIcon covers both UIWindow.img/ItemProtector layouts the
// walker must tolerate (Icon-as-canvas and Icon-as-container) plus the absent
// cases, so the seal-badge asset extraction degrades quietly rather than
// panicking on an unexpected WZ shape.
func TestFindItemProtectorIcon(t *testing.T) {
	directIcon := property.NewCanvas("Icon", 16, 16, 0, 100, 4, nil)
	nestedIcon := property.NewCanvas("0", 16, 16, 0, 200, 4, nil)

	tests := []struct {
		name  string
		props []property.Property
		want  *property.CanvasProperty
	}{
		{
			name: "direct canvas",
			props: []property.Property{
				property.NewSub("ItemProtector", []property.Property{
					property.NewString("origin", "x"),
					directIcon,
				}),
			},
			want: directIcon,
		},
		{
			name: "nested icon container",
			props: []property.Property{
				property.NewSub("ItemProtector", []property.Property{
					property.NewSub("Icon", []property.Property{nestedIcon}),
				}),
			},
			want: nestedIcon,
		},
		{
			name:  "no ItemProtector",
			props: []property.Property{property.NewSub("SomethingElse", nil)},
			want:  nil,
		},
		{
			name: "ItemProtector without Icon",
			props: []property.Property{
				property.NewSub("ItemProtector", []property.Property{
					property.NewString("origin", "x"),
				}),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findItemProtectorIcon(tt.props); got != tt.want {
				t.Fatalf("findItemProtectorIcon = %v, want %v", got, tt.want)
			}
		})
	}
}
