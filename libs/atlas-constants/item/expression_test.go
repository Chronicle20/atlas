package item_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestIsExtraExpressionEmote(t *testing.T) {
	cases := []struct {
		name  string
		emote uint32
		want  bool
	}{
		{"zero", 0, false},
		{"base emote upper bound", 7, false},
		{"first extra", 8, true},
		{"last extra in v83 data", 22, true},
		{"gated upper bound", 23, true},
		{"above client cap", 24, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := item.IsExtraExpressionEmote(tc.emote); got != tc.want {
				t.Errorf("IsExtraExpressionEmote(%d) = %t, want %t", tc.emote, got, tc.want)
			}
		})
	}
}

func TestExtraExpressionItemId(t *testing.T) {
	cases := []struct {
		name   string
		emote  uint32
		wantId item.Id
		wantOk bool
	}{
		{"emote 8 maps to Queasy", 8, item.Id(5160000), true},
		{"emote 22 maps to the last v83 item", 22, item.Id(5160014), true},
		{"emote 23 maps to an id no character can own", 23, item.Id(5160015), true},
		{"base emote is not gated", 7, item.Id(0), false},
		{"above client cap", 24, item.Id(0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotId, gotOk := item.ExtraExpressionItemId(tc.emote)
			if gotId != tc.wantId || gotOk != tc.wantOk {
				t.Errorf("ExtraExpressionItemId(%d) = %d, %t, want %d, %t", tc.emote, gotId, gotOk, tc.wantId, tc.wantOk)
			}
		})
	}
}

func TestExtraExpressionItemIdClassification(t *testing.T) {
	for _, emote := range []uint32{8, 22} {
		id, ok := item.ExtraExpressionItemId(emote)
		if !ok {
			t.Fatalf("ExtraExpressionItemId(%d) ok = false, want true", emote)
		}
		if got := item.GetClassification(id); got != item.ClassificationExpression {
			t.Errorf("ExtraExpressionItemId(%d) = %d, want %d", emote, got, item.ClassificationExpression)
		}
	}
}
