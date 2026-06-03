package atlaspacket

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestRegistryFindsCharacterListEntry(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reg.HasType("CharacterListEntry") {
		t.Error("registry should know CharacterListEntry")
	}
	if calls, ok := reg.Calls("CharacterListEntry"); !ok || len(calls) == 0 {
		t.Errorf("expected Calls('CharacterListEntry') non-empty; got %d / %v", len(calls), ok)
	}
	if !reg.HasType("WorldRecommendation") {
		t.Error("registry should know WorldRecommendation (Write method)")
	}
	if calls, ok := reg.Calls("WorldRecommendation"); !ok || len(calls) != 2 {
		t.Errorf("WorldRecommendation has Write with int32+string = 2 calls; got ok=%v len=%d", ok, len(calls))
	}
}

func TestRegistryFieldTypeStrips(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, _ := NewTypeRegistry(root)
	// CharacterList.characters is []model.CharacterListEntry
	if ft, ok := reg.FieldType("CharacterList", "characters"); !ok || ft != "CharacterListEntry" {
		t.Errorf("FieldType(CharacterList, characters) = (%q, %v); want CharacterListEntry", ft, ok)
	}
}

func TestRegistryDiscoversEncodeForeign(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	// CharacterTemporaryStat has both Encode and EncodeForeign; the registry must
	// expose calls for the EncodeForeign variant under a distinct key.
	if _, ok := reg.Calls("CharacterTemporaryStat::EncodeForeign"); !ok {
		t.Errorf("expected calls registered for CharacterTemporaryStat::EncodeForeign; got none")
	}
	// Encode entry must still resolve under the bare type name.
	if _, ok := reg.Calls("CharacterTemporaryStat"); !ok {
		t.Errorf("expected calls registered for CharacterTemporaryStat (Encode); got none")
	}
}

func TestRegistryRegistersCharacterSubStructs(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	// AttackInfo is intentionally absent — it is a decode-only (serverbound) type
	// with no Encode method, so registry pass-2 cannot register it. Phase 2 Task 12
	// (serverbound hot bucket) will exercise the decode path through whatever
	// mechanism applies; the pipeline does not need AttackInfo registered as a
	// recurse target for clientbound encoders.
	for _, name := range []string{"Pet", "DamageTakenInfo"} {
		if !reg.HasType(name) {
			t.Errorf("registry missing type %s", name)
			continue
		}
		calls, ok := reg.Calls(name)
		if !ok || len(calls) == 0 {
			t.Errorf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
		}
	}
}

func TestRegistryRegistersMovementElements(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	// Top-level wrapper.
	if !reg.HasType("Movement") {
		t.Fatal("registry missing Movement")
	}
	// Element sub-types — each has its own Encode method.
	for _, name := range []string{
		"Element",
		"NormalElement",
		"TeleportElement",
		"StartFallDownElement",
		"FlyingBlockElement",
		"JumpElement",
		"StatChangeElement",
	} {
		if !reg.HasType(name) {
			t.Errorf("registry missing movement element type %s", name)
		}
	}
}

func TestRegistryRegistersCombatSubStructs(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"MonsterModel",
		"MonsterTemporaryStat",
		"MultiTargetForBall",
		"RandTimeForAreaAttack",
	} {
		if !reg.HasType(name) {
			t.Errorf("registry missing combat sub-struct %s", name)
			continue
		}
		calls, ok := reg.Calls(name)
		if !ok || len(calls) == 0 {
			t.Errorf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
		}
	}
}

// TestRegistryQualifiedKeysResolveCollisions verifies the registry stores
// all variants of a colliding short struct name (e.g. Spawn appears in 5
// packages) under qualified keys so per-package field-type resolution and
// per-package Calls lookup do not last-write-wins each other.
func TestRegistryQualifiedKeysResolveCollisions(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	// All five Spawn variants must be present under their qualified keys.
	for _, qual := range []string{
		"monster/clientbound.Spawn",
		"drop/clientbound.Spawn",
		"reactor/clientbound.Spawn",
		"npc/clientbound.Spawn",
		"pet/serverbound.Spawn",
	} {
		if !reg.HasType(qual) {
			t.Errorf("registry missing qualified Spawn variant %q", qual)
			continue
		}
		if calls, ok := reg.Calls(qual); !ok || len(calls) == 0 {
			t.Errorf("Calls(%q) returned no calls (ok=%v len=%d)", qual, ok, len(calls))
		}
	}
	// And the ambiguous short name "Spawn" must NOT return a Calls list — the
	// caller has to disambiguate via Qualify or pass the qualified key.
	if calls, ok := reg.Calls("Spawn"); ok && calls != nil {
		t.Errorf("Calls(\"Spawn\") on ambiguous short name should return (nil,false); got (%d calls, ok=%v)", len(calls), ok)
	}
}

// TestRegistryQualifyPrefersSamePackage verifies that Qualify resolves
// ambiguous short names by preferring the caller-provided context package
// over arbitrary first-match selection.
func TestRegistryQualifyPrefersSamePackage(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		short, pkg, want string
	}{
		{"Spawn", "monster/clientbound", "monster/clientbound.Spawn"},
		{"Spawn", "drop/clientbound", "drop/clientbound.Spawn"},
		{"Spawn", "reactor/clientbound", "reactor/clientbound.Spawn"},
		{"Destroy", "monster/clientbound", "monster/clientbound.Destroy"},
		{"Destroy", "drop/clientbound", "drop/clientbound.Destroy"},
		{"Movement", "monster/clientbound", "monster/clientbound.Movement"},
		{"Movement", "pet/clientbound", "pet/clientbound.Movement"},
		{"Movement", "model", "model.Movement"},
	}
	for _, c := range cases {
		got := reg.Qualify(c.short, c.pkg)
		if got != c.want {
			t.Errorf("Qualify(%q, %q) = %q; want %q", c.short, c.pkg, got, c.want)
		}
	}
}

func TestRegistryRegistersSocialSubStructs(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"GuildMember", "Buddy", "Avatar"} {
		if !reg.HasType(name) {
			t.Errorf("registry missing type %s", name)
			continue
		}
		calls, ok := reg.Calls(name)
		if !ok || len(calls) == 0 {
			t.Errorf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
		}
	}
}

func TestRegistryStillRegistersMovementAfterCombatExtension(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"Movement",
		"Element",
		"NormalElement",
		"TeleportElement",
		"StartFallDownElement",
		"FlyingBlockElement",
		"JumpElement",
		"StatChangeElement",
	} {
		if !reg.HasType(name) {
			t.Errorf("registry missing movement sub-type %s (task-028 regression)", name)
		}
	}
}

func TestRegistryRegistersNpcShopItem(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	name := "ShopCommodity" // npc/clientbound.ShopCommodity in shop_list.go
	if !reg.HasType(name) {
		t.Fatalf("registry missing type %s", name)
	}
	// ShopCommodity has no Encode/Write/EncodeEntry/EncodeBytes method of its own;
	// its fields are encoded inline inside ShopList.Encode — the registry therefore
	// returns no Calls for this type (known registry limitation: inline sub-structs
	// without their own encode method are not resolvable as KindRecurse targets).
	calls, ok := reg.Calls(name)
	if ok && len(calls) > 0 {
		// If a future change adds an Encode method to ShopCommodity the inline
		// encoding limitation is resolved — record this as an unexpected pass so
		// the comment above can be removed.
		t.Logf("UNEXPECTED: %s.Encode produced calls (ok=%v len=%d) — inline-encoding concern is resolved", name, ok, len(calls))
	}
}

func TestRegistryRegistersCommerceSubStructs(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"CashInventoryItem", // EncodeBytes (flat)
		"AddEntry",          // EncodeEntry (closure)
		"QuantityUpdateEntry",
		"MoveEntry",
		"RemoveEntry",
	} {
		if !reg.HasType(name) {
			t.Errorf("registry missing type %s", name)
			continue
		}
		calls, ok := reg.Calls(name)
		if !ok || len(calls) == 0 {
			t.Errorf("%s produced no calls (ok=%v len=%d)", name, ok, len(calls))
		}
	}
}

// TestRegistryRegistersNpcConversation asserts the registry covers the NPC
// conversation encoder. npc/clientbound/conversation.go has one top-level
// struct NpcConversation (with Encode) plus multiple per-dialog-type
// *ConversationDetail structs each with their own Encode method.
func TestRegistryRegistersNpcConversation(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	// Top-level wrapper: single struct with constructor NewNpcConversation + Encode.
	name := "NpcConversation"
	if !reg.HasType(name) {
		t.Fatalf("registry missing %s", name)
	}
	calls, ok := reg.Calls(name)
	if !ok || len(calls) == 0 {
		t.Fatalf("%s.Encode produced no calls (ok=%v len=%d)", name, ok, len(calls))
	}
	// Per-dialog-type detail structs — each has its own Encode method.
	for _, detail := range []string{
		"SayConversationDetail",
		"SayImageConversationDetail",
		"AskYesNoConversationDetail",
		"AskTextConversationDetail",
		"AskNumberConversationDetail",
		"AskMenuConversationDetail",
		"AskQuizConversationDetail",
		"AskSpeedQuizConversationDetail",
		"AskAvatarConversationDetail",
		"AskMemberShopAvatarConversationDetail",
		"AskPetConversationDetail",
		"AskPetAllConversationDetail",
		"AskBoxTextConversationDetail",
		"AskSlideMenuConversationDetail",
	} {
		if !reg.HasType(detail) {
			t.Errorf("registry missing conversation detail type %s", detail)
			continue
		}
		dc, dok := reg.Calls(detail)
		if !dok || len(dc) == 0 {
			t.Errorf("%s.Encode produced no calls (ok=%v len=%d)", detail, dok, len(dc))
		}
	}
}
