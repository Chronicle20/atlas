package expiration_test

import (
	"atlas-asset-expiration/expiration"
	"testing"
)

func TestIsReapable(t *testing.T) {
	cases := []struct {
		name       string
		templateId uint32
		want       bool
	}{
		{"equip", 1002357, true},
		{"consumable", 2000000, true},
		{"etc", 4000000, true},
		{"pet (dog)", 5000000, false},
		{"pet (high id in 500)", 5009999, false},
		{"cash, not a pet (character effect)", 5010000, true},
		{"water of life", 5180000, true},
	}
	for _, c := range cases {
		if got := expiration.IsReapable(c.templateId); got != c.want {
			t.Errorf("%s: IsReapable(%d) = %v, want %v", c.name, c.templateId, got, c.want)
		}
	}
}
