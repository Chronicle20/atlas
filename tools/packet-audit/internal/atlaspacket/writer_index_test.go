package atlaspacket

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestTypeForWriterConstBacked verifies that TypeForWriter resolves a const-backed
// Operation() pattern (WriterName differs from struct name, like MonsterStatSet).
func TestTypeForWriterConstBacked(t *testing.T) {
	dir := setupWriterTestDir(t, map[string]string{
		"pkg/writer_const.go": readTestdata(t, "writer_name_const.go.txt"),
	})
	reg, err := NewTypeRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}

	// SomeStruct.Operation() returns SomeWriter = "SomePacket"
	qual, ok := reg.TypeForWriter("pkg", "SomePacket")
	if !ok {
		t.Fatal("TypeForWriter(pkg, SomePacket) = miss; want pkg.SomeStruct")
	}
	if qual != "pkg.SomeStruct" {
		t.Errorf("TypeForWriter(pkg, SomePacket) = %q; want pkg.SomeStruct", qual)
	}

	// OtherStruct.Operation() returns OtherWriter = "OtherPacket"
	qual2, ok2 := reg.TypeForWriter("pkg", "OtherPacket")
	if !ok2 {
		t.Fatal("TypeForWriter(pkg, OtherPacket) = miss; want pkg.OtherStruct")
	}
	if qual2 != "pkg.OtherStruct" {
		t.Errorf("TypeForWriter(pkg, OtherPacket) = %q; want pkg.OtherStruct", qual2)
	}
}

// TestTypeForWriterDirectLiteral verifies that TypeForWriter resolves an
// Operation() that returns a string literal directly (serverbound handle pattern).
func TestTypeForWriterDirectLiteral(t *testing.T) {
	dir := setupWriterTestDir(t, map[string]string{
		"pkg/writer_literal.go": readTestdata(t, "writer_name_literal.go.txt"),
	})
	reg, err := NewTypeRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}

	qual, ok := reg.TypeForWriter("pkg", "DirectLiteralHandle")
	if !ok {
		t.Fatal("TypeForWriter(pkg, DirectLiteralHandle) = miss; want pkg.DirectLiteral")
	}
	if qual != "pkg.DirectLiteral" {
		t.Errorf("TypeForWriter(pkg, DirectLiteralHandle) = %q; want pkg.DirectLiteral", qual)
	}
}

// TestTypeForWriterMiss verifies that TypeForWriter returns false when no
// struct in the given package has an Operation() returning the requested name.
func TestTypeForWriterMiss(t *testing.T) {
	dir := setupWriterTestDir(t, map[string]string{
		"pkg/writer_const.go": readTestdata(t, "writer_name_const.go.txt"),
	})
	reg, err := NewTypeRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Completely unknown writer name.
	if _, ok := reg.TypeForWriter("pkg", "NoSuchWriter"); ok {
		t.Error("TypeForWriter for unknown name should return (_, false)")
	}
	// Known writer name but wrong package directory.
	if _, ok := reg.TypeForWriter("other/pkg", "SomePacket"); ok {
		t.Error("TypeForWriter with wrong package should return (_, false)")
	}
}

// TestTypeForWriterRealTree verifies the index against the real atlas-packet tree
// for the three canonical opaque-consumer patterns (Task 3.2).
func TestTypeForWriterRealTree(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		pkgDir     string
		writerName string
		wantKey    string
	}{
		// monster/clientbound: WriterName "MonsterStatSet" → struct StatSet
		{"monster/clientbound", "MonsterStatSet", "monster/clientbound.StatSet"},
		// monster/clientbound: WriterName "MoveMonster" → struct Movement
		{"monster/clientbound", "MoveMonster", "monster/clientbound.Movement"},
		// buddy/clientbound: WriterName "BuddyInvite" → struct Invite
		{"buddy/clientbound", "BuddyInvite", "buddy/clientbound.Invite"},
		// npc/clientbound: WriterName "NPCConversation" → struct NpcConversation
		{"npc/clientbound", "NPCConversation", "npc/clientbound.NpcConversation"},
		// serverbound direct-literal: "OperationExpel" → struct OperationExpel
		{"party/serverbound", "OperationExpel", "party/serverbound.OperationExpel"},
		// serverbound const-backed handle: "NPCStartConversationHandle" → StartConversation
		{"npc/serverbound", "NPCStartConversationHandle", "npc/serverbound.StartConversation"},
	}
	for _, c := range cases {
		qual, ok := reg.TypeForWriter(c.pkgDir, c.writerName)
		if !ok {
			t.Errorf("TypeForWriter(%q, %q) = miss; want %q", c.pkgDir, c.writerName, c.wantKey)
			continue
		}
		if qual != c.wantKey {
			t.Errorf("TypeForWriter(%q, %q) = %q; want %q", c.pkgDir, c.writerName, qual, c.wantKey)
		}
	}
}

// TestTypeForWriterOpaqueRecursion verifies that transitiveRecurse (via the
// real registry) resolves opaque-consumer packets to their opaque sub-struct
// types. This proves Commit-1 closes the WriterName→StructName gap that made
// IsTier1 fail for opaque consumers (Task 3.2).
func TestTypeForWriterOpaqueRecursion(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "libs", "atlas-packet")
	reg, err := NewTypeRegistry(root)
	if err != nil {
		t.Fatal(err)
	}

	// monster/clientbound/MonsterStatSet must recurse into MonsterTemporaryStat (opaque).
	types := testTransitiveRecurseTypes(reg, "monster/clientbound/MonsterStatSet")
	if !containsAny(types, "MonsterTemporaryStat") {
		t.Errorf("monster/clientbound/MonsterStatSet recursion = %v; want MonsterTemporaryStat", types)
	}

	// monster/clientbound/MoveMonster must recurse into MultiTargetForBall.
	types2 := testTransitiveRecurseTypes(reg, "monster/clientbound/MoveMonster")
	if !containsAny(types2, "MultiTargetForBall") {
		t.Errorf("monster/clientbound/MoveMonster recursion = %v; want MultiTargetForBall", types2)
	}

	// buddy/clientbound/BuddyInvite must recurse into Buddy (the sub-struct it encodes).
	types3 := testTransitiveRecurseTypes(reg, "buddy/clientbound/BuddyInvite")
	if !containsAny(types3, "Buddy") {
		t.Errorf("buddy/clientbound/BuddyInvite recursion = %v; want Buddy", types3)
	}
}

// helpers

// testTransitiveRecurseTypes is a local copy of the cmd.transitiveRecurseTypes
// logic, placed here so the atlaspacket package tests don't import cmd.
func testTransitiveRecurseTypes(reg *TypeRegistry, packetID string) []string {
	i := len(packetID) - 1
	for i >= 0 && packetID[i] != '/' {
		i--
	}
	if i < 0 {
		return nil
	}
	pkgPath := packetID[:i]
	writerName := packetID[i+1:]
	qualKey, ok := reg.TypeForWriter(pkgPath, writerName)
	if !ok {
		qualKey = pkgPath + "." + writerName
	}
	seen := map[string]bool{}
	var walk func(key string)
	walk = func(key string) {
		if seen[key] {
			return
		}
		seen[key] = true
		calls, callOk := reg.Calls(key)
		if !callOk {
			return
		}
		for _, c := range calls {
			if c.Kind == KindRecurse && c.RecurseType != "" {
				walk(c.RecurseType)
			}
		}
	}
	walk(qualKey)
	delete(seen, qualKey)
	var out []string
	for k := range seen {
		out = append(out, k)
	}
	extra := make([]string, 0, len(out))
	for _, k := range out {
		if j := lastIndex(k, '.'); j >= 0 {
			short := k[j+1:]
			if len(short) > 16 && short[len(short)-16:] == "::EncodeForeign" {
				short = short[:len(short)-16]
			}
			extra = append(extra, short)
		}
	}
	return append(out, extra...)
}

func lastIndex(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func containsAny(slice []string, sub string) bool {
	for _, s := range slice {
		if s == sub || len(s) >= len(sub) && containsStr(s, sub) {
			return true
		}
	}
	return false
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// setupWriterTestDir creates a temp directory tree from a map of
// relative path → file content. Returns the root temp directory.
func setupWriterTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		dst := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// readTestdata reads a testdata/*.go.txt fixture and returns its content as a
// string so it can be used in setupWriterTestDir. The fixture package name
// "fixture" must be used — we rename it so NewTypeRegistry parses it as a valid
// Go file in the requested package dir.
func readTestdata(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("readTestdata(%q): %v", name, err)
	}
	return string(raw)
}
