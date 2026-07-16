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

func TestIsFreeMarketRoom(t *testing.T) {
	cases := []struct {
		name string
		id   Id
		want bool
	}{
		{"below range", Id(909999999), false},
		{"FM entrance", Id(910000000), true},
		{"FM room 1", Id(910000001), true},
		{"FM room 22 (last)", Id(910000022), true},
		{"above range", Id(910000023), false},
		{"henesys (unrelated)", Id(100000000), false},
		{"zero", Id(0), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsFreeMarketRoom(c.id); got != c.want {
				t.Errorf("IsFreeMarketRoom(%d) = %v, want %v", c.id, got, c.want)
			}
		})
	}
}
