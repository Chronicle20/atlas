package redis

import (
	"context"
	"testing"
)

func TestTenantKeyedHash_SetNXAndDeleteKey(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	h := NewTenantKeyedHash[string](client, "reactor:spot", func(k string) string { return k })
	tm := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	mapKey := "0:1:100:nil"

	first, err := h.SetNX(ctx, tm, mapKey, "5000:10:20", "1")
	if err != nil || !first {
		t.Fatalf("first SetNX = %v,%v want true,nil", first, err)
	}
	wantKey := "atlas:reactor:spot:" + TenantKey(tm) + ":" + mapKey
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	second, _ := h.SetNX(ctx, tm, mapKey, "5000:10:20", "1")
	if second {
		t.Fatalf("second SetNX on same field must be false")
	}
	_ = h.Set(ctx, tm, mapKey, "5000:30:40", "1")
	all, _ := h.GetAll(ctx, tm, mapKey)
	if len(all) != 2 {
		t.Fatalf("GetAll = %v want len 2", all)
	}
	_ = h.Del(ctx, tm, mapKey, "5000:10:20")
	if ok, _ := h.Exists(ctx, tm, mapKey, "5000:10:20"); ok {
		t.Fatalf("field should be gone")
	}
	_ = h.DeleteKey(ctx, tm, mapKey)
	all, _ = h.GetAll(ctx, tm, mapKey)
	if len(all) != 0 {
		t.Fatalf("DeleteKey should empty the hash; got %v", all)
	}
}
