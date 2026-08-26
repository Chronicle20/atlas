package wzdiff

import "testing"

func TestNodePath(t *testing.T) {
	tests := []struct {
		name  string
		chain []Node
		want  string
	}{
		{
			name:  "single container",
			chain: []Node{{Kind: "imgdir", Name: "0"}},
			want:  "/imgdir:0",
		},
		{
			name: "nested",
			chain: []Node{
				{Kind: "imgdir", Name: "0"},
				{Kind: "imgdir", Name: "event"},
				{Kind: "int", Name: "state"},
			},
			want: "/imgdir:0/imgdir:event/int:state",
		},
		{
			name: "canvas child",
			chain: []Node{
				{Kind: "imgdir", Name: "1"},
				{Kind: "canvas", Name: "0"},
				{Kind: "vector", Name: "origin"},
			},
			want: "/imgdir:1/canvas:0/vector:origin",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathOf(tc.chain...); got != tc.want {
				t.Errorf("pathOf(%v) = %q, want %q", tc.chain, got, tc.want)
			}
		})
	}
}

func TestDeltaString(t *testing.T) {
	tests := []struct {
		name  string
		delta Delta
		want  string
	}{
		{
			name:  "int",
			delta: Delta{Path: "/imgdir:0/int:state", Attrs: "value=1"},
			want:  "      /imgdir:0/int:state | value=1",
		},
		{
			name:  "container",
			delta: Delta{Path: "/imgdir:0/imgdir:event", Attrs: ""},
			want:  "      /imgdir:0/imgdir:event | ",
		},
		{
			name:  "canvas",
			delta: Delta{Path: "/imgdir:1/canvas:0", Attrs: "height=121 width=100"},
			want:  "      /imgdir:1/canvas:0 | height=121 width=100",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.delta.String(); got != tc.want {
				t.Errorf("Delta.String() = %q, want %q", got, tc.want)
			}
		})
	}
}
