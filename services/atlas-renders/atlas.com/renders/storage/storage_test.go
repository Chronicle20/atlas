package storage

import "testing"

func TestNewCachesAllocates(t *testing.T) {
	c := NewCaches(8, 8, 8)
	if c.Atlas == nil || c.Map == nil || c.Scope == nil {
		t.Fatal("nil cache")
	}
}
