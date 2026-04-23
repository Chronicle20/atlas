package main

import "testing"

func TestSelect_DirectLibChange(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs: []string{"lib-c"},
	})
	if !equalSet(sel.Services, []string{"svc-b"}) {
		t.Errorf("services=%v want [svc-b]", sel.Services)
	}
	if !equalSet(sel.Libs, []string{"lib-c"}) {
		t.Errorf("libs=%v want [lib-c]", sel.Libs)
	}
}

func TestSelect_TransitiveLibChange(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs: []string{"lib-a"},
	})
	// svc-a → lib-b → lib-a, so svc-a is affected.
	// lib-b → lib-a, so lib-b is affected.
	if !equalSet(sel.Services, []string{"svc-a"}) {
		t.Errorf("services=%v want [svc-a]", sel.Services)
	}
	if !equalSet(sel.Libs, []string{"lib-a", "lib-b"}) {
		t.Errorf("libs=%v want [lib-a lib-b]", sel.Libs)
	}
}

func TestSelect_ChangedServiceUnion(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs:     []string{"lib-c"},
		ChangedServices: []string{"svc-a"},
	})
	if !equalSet(sel.Services, []string{"svc-a", "svc-b"}) {
		t.Errorf("services=%v want [svc-a svc-b]", sel.Services)
	}
}

func TestSelect_NoChanges(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{})
	if len(sel.Services) != 0 || len(sel.Libs) != 0 {
		t.Errorf("expected empty selection, got services=%v libs=%v", sel.Services, sel.Libs)
	}
}

func TestSelect_ForceAll(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{ForceAll: true})
	if !equalSet(sel.Services, []string{"svc-a", "svc-b"}) {
		t.Errorf("services=%v want [svc-a svc-b]", sel.Services)
	}
	if !equalSet(sel.Libs, []string{"lib-a", "lib-b", "lib-c"}) {
		t.Errorf("libs=%v want [lib-a lib-b lib-c]", sel.Libs)
	}
}

func TestSelect_UnknownNameIgnored(t *testing.T) {
	g, err := BuildGraph("testdata/transitive")
	if err != nil {
		t.Fatal(err)
	}
	sel := Select(g, SelectInput{
		ChangedLibs:     []string{"no-such-lib"},
		ChangedServices: []string{"no-such-svc"},
	})
	if len(sel.Services) != 0 || len(sel.Libs) != 0 {
		t.Errorf("unknown names should select nothing, got services=%v libs=%v", sel.Services, sel.Libs)
	}
}
