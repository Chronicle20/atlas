package wz

import (
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// posCountingFile wraps a readSeekerAt and counts calls shaped exactly like
// Reader.Pos() — Seek(0, io.SeekCurrent) — the file-level signature of every
// f.Seek syscall Pos() performs. Used by
// TestTraceNilHookCostsNoExtraPosCalls to prove the nil-hook path performs
// no extra Pos() calls without a test-only field on the production Reader
// type (task-262 M1-3).
type posCountingFile struct {
	readSeekerAt
	posCalls int64
}

// This counting is exact by construction, not by coincidence: Reader.Pos()
// is the only Reader method that issues Seek(0, io.SeekCurrent) against the
// underlying file. Reader.Skip(0) does NOT reach here — it short-circuits
// on its own zero guard in reader.go before ever calling f.Seek — so this
// wrapper cannot conflate a degenerate Skip(0) with a real Pos() call
// (task-262 fix round 1).
func (p *posCountingFile) Seek(offset int64, whence int) (int64, error) {
	if offset == 0 && whence == io.SeekCurrent {
		p.posCalls++
	}
	return p.readSeekerAt.Seek(offset, whence)
}

// openCounting opens path the same way Open does, but routes the reader
// through a posCountingFile so the caller can read the resulting Pos() call
// count.
func openCounting(t *testing.T, path string) (*File, *posCountingFile) {
	t.Helper()
	osFile, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture file: %v", err)
	}
	counting := &posCountingFile{readSeekerAt: osFile}
	f, err := openWithReader(logrus.StandardLogger(), path, osFile, newReaderWithSeeker(counting))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	return f, counting
}

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
	propOrderOK := true
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
		if ev.Kind != "sub" && (ev.DeclaredEnd != 0 || ev.ActualEnd != 0) {
			t.Errorf("non-sub event %+v has non-zero DeclaredEnd/ActualEnd", ev)
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

// TestSetTraceEmitsSubEventDeclaredActualEnd proves the Kind: "sub" event
// populates DeclaredEnd/ActualEnd (matching endPos/actualEnd baked into
// Detail) so a gate can read them as structured fields instead of parsing
// the Detail string (task-262 R2).
func TestSetTraceEmitsSubEventDeclaredActualEnd(t *testing.T) {
	path := writeFixture(t, gmsFixtureBuilder(), "ItemSubEnd.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var events []TraceEvent
	f.SetTrace(func(ev TraceEvent) {
		events = append(events, ev)
	})

	imgs := f.Root().Directories()[0].Images()
	if _, err := imgs[0].Properties(); err != nil {
		t.Fatalf("properties: %v", err)
	}

	var subEvent *TraceEvent
	for i, ev := range events {
		if ev.Path == "/info" && ev.Kind == "sub" {
			subEvent = &events[i]
		}
	}
	if subEvent == nil {
		t.Fatalf("no event with Path=/info Kind=sub in %+v", events)
	}
	if subEvent.DeclaredEnd == 0 {
		t.Errorf("sub event DeclaredEnd = 0, want the declared-size end position: %+v", *subEvent)
	}
	if subEvent.ActualEnd == 0 {
		t.Errorf("sub event ActualEnd = 0, want the actual decode end position: %+v", *subEvent)
	}
	if subEvent.DeclaredEnd != subEvent.ActualEnd {
		t.Errorf("sub event DeclaredEnd=%d ActualEnd=%d, want them equal for a well-formed fixture", subEvent.DeclaredEnd, subEvent.ActualEnd)
	}
	wantDetail := fmt.Sprintf("declaredSize=%d endPos=%d actualEnd=%d",
		subEvent.DeclaredEnd-subEvent.StartOff-4, subEvent.DeclaredEnd, subEvent.ActualEnd)
	if subEvent.Detail != wantDetail {
		t.Errorf("sub event Detail = %q, want %q", subEvent.Detail, wantDetail)
	}

	for _, ev := range events {
		if ev.Kind == "sub" {
			continue
		}
		if ev.DeclaredEnd != 0 || ev.ActualEnd != 0 {
			t.Errorf("non-sub event %+v has non-zero DeclaredEnd/ActualEnd", ev)
		}
	}
}

// wantNoHookPosCalls is the exact number of Reader.Pos() calls (each one an
// f.Seek syscall, counted by posCountingFile) a fully correct parse of
// gmsFixtureBuilder's image makes with no trace hook installed: the handful
// of calls parsing genuinely needs regardless of tracing (e.g. computing a
// type-9 sub-object's endPos for the recovery reseek), and none of the
// trace-only position captures, which must all be gated behind
// "hook != nil". Recompute by temporarily logging the posCountingFile's
// posCalls in a no-hook parse of this fixture if the fixture or the
// parser's required Pos() calls change.
const wantNoHookPosCalls = 8

// TestTraceNilHookCostsNoExtraPosCalls proves the nil-hook path performs
// exactly the Pos() calls parsing structurally requires and no more — i.e.
// every trace-only Pos() capture is gated behind "hook != nil" and costs
// nothing when unset. It also confirms installing a hook measurably adds
// calls, so the exact-count assertion isn't vacuously satisfied by a parser
// that never calls Pos() for tracing at all. If a future change hoists a
// gated capture above its "hook != nil" check (as commit f780b23ad did),
// noHookCalls rises above wantNoHookPosCalls and this test fails
// (task-262 fix round 1).
func TestTraceNilHookCostsNoExtraPosCalls(t *testing.T) {
	builder := gmsFixtureBuilder()

	pathNoHook := writeFixture(t, builder, "ItemNoHook.wz")
	fNoHook, countingNoHook := openCounting(t, pathNoHook)
	defer fNoHook.Close()

	imgsNoHook := fNoHook.Root().Directories()[0].Images()
	if _, err := imgsNoHook[0].Properties(); err != nil {
		t.Fatalf("properties (no hook): %v", err)
	}
	noHookCalls := countingNoHook.posCalls

	if noHookCalls != wantNoHookPosCalls {
		t.Fatalf("nil-hook parse made %d Pos() calls, want exactly %d — a trace-only position capture is running unconditionally on the nil-hook path", noHookCalls, wantNoHookPosCalls)
	}

	pathHook := writeFixture(t, builder, "ItemHook.wz")
	fHook, countingHook := openCounting(t, pathHook)
	defer fHook.Close()

	fHook.SetTrace(func(TraceEvent) {})

	imgsHook := fHook.Root().Directories()[0].Images()
	if _, err := imgsHook[0].Properties(); err != nil {
		t.Fatalf("properties (hook): %v", err)
	}
	hookCalls := countingHook.posCalls

	if hookCalls <= noHookCalls {
		t.Fatalf("hook-installed parse made %d Pos() calls, no-hook parse made %d — expected strictly more with a hook installed (otherwise this fixture can't distinguish the two paths)", hookCalls, noHookCalls)
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

// TestSkipZeroPerformsNoSeek proves Skip(0) is a no-op that never reaches
// the underlying file's Seek — the counterpart to Skip's zero guard in
// reader.go. Without the guard, Skip(0) would issue Seek(0, io.SeekCurrent),
// the exact same file-level call shape Pos() emits, making
// TestTraceNilHookCostsNoExtraPosCalls' posCountingFile count Skip(0) calls
// as if they were Pos() calls (task-262 fix round 1).
func TestSkipZeroPerformsNoSeek(t *testing.T) {
	path := writeFixture(t, gmsFixtureBuilder(), "ItemSkipZero.wz")
	f, counting := openCounting(t, path)
	defer f.Close()

	before := counting.posCalls
	if err := f.reader.Skip(0); err != nil {
		t.Fatalf("Skip(0): %v", err)
	}
	if counting.posCalls != before {
		t.Fatalf("Skip(0) changed posCalls from %d to %d — it must not reach the underlying Seek", before, counting.posCalls)
	}
}

// TestSetTraceEmitsUOLEvent proves the UOL extended-property branch emits
// its own Kind: "uol" TraceEvent bracketing the bytes past the shared
// Kind: "extended" tag event (the 1-byte skip plus the target string
// block), so a divergence inside a UOL value is visible the same way it
// already is for canvas and sub-object values (task-262 M1-2).
func TestSetTraceEmitsUOLEvent(t *testing.T) {
	builder := wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionGMS).
		AddDir(wztest.Dir{
			Name: "Item",
			Images: []wztest.Image{
				wztest.Img("0200",
					wztest.Sub("info",
						wztest.UOL("link", "../other/path"),
					),
				),
			},
		})
	path := writeFixture(t, builder, "ItemUOL.wz")

	f, err := Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var events []TraceEvent
	f.SetTrace(func(ev TraceEvent) {
		events = append(events, ev)
	})

	imgs := f.Root().Directories()[0].Images()
	if _, err := imgs[0].Properties(); err != nil {
		t.Fatalf("properties: %v", err)
	}

	var uolEvent *TraceEvent
	for i, ev := range events {
		if ev.Path == "/info/link" && ev.Kind == "uol" {
			uolEvent = &events[i]
		}
	}
	if uolEvent == nil {
		t.Fatalf("no event with Path=/info/link Kind=uol in %+v", events)
	}
	if uolEvent.Detail != "../other/path" {
		t.Errorf("uol event Detail = %q, want %q", uolEvent.Detail, "../other/path")
	}
	if uolEvent.EndOff <= uolEvent.StartOff {
		t.Errorf("uol event has EndOff <= StartOff: %+v", uolEvent)
	}
}
