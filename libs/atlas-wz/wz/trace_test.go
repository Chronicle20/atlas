package wz

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

func gmsFixtureBuilder() *wztest.Builder {
	return wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionGMS).
		AddDir(wztest.Dir{
			Name: "Item",
			Images: []wztest.Image{
				wztest.Img("0200",
					wztest.Sub("info",
						wztest.Int("state", 1),
						wztest.Str("name", "x"),
					),
				),
			},
		})
}

// TestSetTraceEmitsNodeEvents proves the trace hook records the decode of
// every node in the property tree with offsets that bracket what was
// actually consumed — the mechanism that turns "the tree is wrong" into
// "the stream diverged at offset X" (task-262 T2).
func TestSetTraceEmitsNodeEvents(t *testing.T) {
	path := writeFixture(t, gmsFixtureBuilder(), "Item.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var events []TraceEvent
	f.SetTrace(func(ev TraceEvent) {
		events = append(events, ev)
	})

	dirs := f.Root().Directories()
	if len(dirs) != 1 || dirs[0].Name() != "Item" {
		t.Fatalf("root dirs = %+v, want one dir Item", dirs)
	}
	imgs := dirs[0].Images()
	if len(imgs) != 1 || imgs[0].Name() != "0200" {
		t.Fatalf("images = %+v, want one image 0200", imgs)
	}

	if _, err := imgs[0].Properties(); err != nil {
		t.Fatalf("properties: %v", err)
	}

	if len(events) == 0 {
		t.Fatalf("expected trace events, got none")
	}

	var haveExtendedInfo, havePropState, havePropName, haveStringblock bool
	var lastPropStart int64 = -1
	var propOrderOK = true
	for _, ev := range events {
		if ev.EndOff < ev.StartOff {
			t.Fatalf("event %+v has EndOff < StartOff", ev)
		}
		if ev.Path == "/info" && ev.Kind == "extended" {
			haveExtendedInfo = true
		}
		if ev.Path == "/info/state" && ev.Kind == "prop" && ev.Type == 3 {
			havePropState = true
		}
		if ev.Path == "/info/name" && ev.Kind == "prop" && ev.Type == 8 {
			havePropName = true
		}
		if ev.Kind == "stringblock" && ev.Detail != "" {
			haveStringblock = true
		}
		if ev.Kind == "prop" {
			if ev.StartOff < lastPropStart {
				propOrderOK = false
			}
			lastPropStart = ev.StartOff
		}
	}

	if !haveExtendedInfo {
		t.Errorf("no event with Path=/info Kind=extended in %+v", events)
	}
	if !havePropState {
		t.Errorf("no event with Path=/info/state Kind=prop Type=3 in %+v", events)
	}
	if !havePropName {
		t.Errorf("no event with Path=/info/name Kind=prop Type=8 in %+v", events)
	}
	if !haveStringblock {
		t.Errorf("no stringblock event with non-empty Detail in %+v", events)
	}
	if !propOrderOK {
		t.Errorf("prop events not emitted in non-decreasing StartOff order: %+v", events)
	}
}

// TestTraceNilByDefault proves a File with no trace installed parses
// exactly as before — no panic, same tree — so the hook is provably
// zero-cost when unused.
func TestTraceNilByDefault(t *testing.T) {
	path := writeFixture(t, gmsFixtureBuilder(), "Item.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	imgs := f.Root().Directories()[0].Images()
	props, err := imgs[0].Properties()
	if err != nil {
		t.Fatalf("properties: %v", err)
	}
	if len(props) != 1 {
		t.Fatalf("top-level props = %d, want 1", len(props))
	}
	sub, ok := findProp(t, props, "info").(*property.SubProperty)
	if !ok {
		t.Fatalf("info is not a SubProperty")
	}
	if len(sub.Children()) != 2 {
		t.Fatalf("info children = %d, want 2", len(sub.Children()))
	}
}

// TestSetTraceOnSubFileDelegatesToParent proves SetTrace called on a parent
// File fires for a sub-file's image parse, and that events are not
// duplicated by double delegation.
func TestSetTraceOnSubFileDelegatesToParent(t *testing.T) {
	path := writeFixture(t, gmsFixtureBuilder(), "Item.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var itemDir *Directory
	for _, d := range f.Root().Directories() {
		if d.Name() == "Item" {
			itemDir = d
		}
	}
	if itemDir == nil {
		t.Fatalf("Item subdirectory not found")
	}
	sub := NewSubFile(f, itemDir, "Item")

	var events []TraceEvent
	f.SetTrace(func(ev TraceEvent) {
		events = append(events, ev)
	})

	imgs := sub.Root().Images()
	if len(imgs) != 1 || imgs[0].Name() != "0200" {
		t.Fatalf("sub images = %+v, want [0200]", imgs)
	}
	if _, err := imgs[0].Properties(); err != nil {
		t.Fatalf("sub image properties: %v", err)
	}

	if len(events) == 0 {
		t.Fatalf("expected trace events fired through sub-file delegation, got none")
	}

	seen := make(map[[3]any]int)
	for _, ev := range events {
		key := [3]any{ev.Path, ev.Kind, ev.StartOff}
		seen[key]++
		if seen[key] > 1 {
			t.Fatalf("event %+v emitted more than once — duplicated by delegation", ev)
		}
	}
}
