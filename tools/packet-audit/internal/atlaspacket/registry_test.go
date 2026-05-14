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
