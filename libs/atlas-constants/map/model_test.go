package _map

import "testing"

func TestIdIsSentinel(t *testing.T) {
	cases := []struct {
		name string
		id   Id
		want bool
	}{
		{"sentinel", EmptyMapId, true},
		{"zero", Id(0), false},
		{"henesys", Id(100000000), false},
		{"kpq lobby", Id(103000890), false},
		{"one below sentinel", Id(999999998), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.id.IsSentinel(); got != c.want {
				t.Fatalf("IsSentinel() = %v, want %v", got, c.want)
			}
		})
	}
}
