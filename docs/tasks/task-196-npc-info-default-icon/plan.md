# NPC `info/default` Icon Precedence — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `ExtractNpcIcon` prefer an NPC's `info/default` canvas over a placeholder `stand/0`, so the 33 NPCs that currently produce a 1-pixel sliver icon get their real likeness.

**Architecture:** Split the NPC canvas finder away from the mob/reactor one. A new `findNpcCanvas` checks for a canvas named `default` among `info`'s children and falls back to the existing `findStandCanvas` otherwise. `ExtractNpcIcon` passes the new finder to the unchanged `extractEntityIcon`, which threads it through link resolution automatically. `ExtractMobIcon` and `ExtractReactorIcon` are not touched.

**Tech Stack:** Go, `libs/atlas-wz` module (own `go.mod`, wired via `go.work`), `wztest` fixture builder, standard `testing`.

## Global Constraints

- Read `docs/tasks/task-196-npc-info-default-icon/context.md` before starting — it documents two structural traps (`Mob.wz` has no `info/default`; `1209003.img` has a *top-level* `default` dir) and the verified fixture mechanics.
- Work in the worktree `.worktrees/task-196-npc-info-default-icon` on branch `task-196-npc-info-default-icon`. Never edit the main repo.
- `findStandCanvas` keeps its current body. `ExtractMobIcon` and `ExtractReactorIcon` keep passing `findStandCanvas`.
- Match only a `*property.CanvasProperty` named `default` among **`info`'s children** — never a sub-property, never a top-level `default` dir.
- No new helper functions beyond `findNpcCanvas`. `findSub` and `findSubCanvas` already exist in `extract.go` and are sufficient.
- No `go.mod` changes. If one becomes necessary, `docker buildx bake atlas-data` becomes mandatory before the branch is called done.
- All fixture canvases are 1×1 format 2 (`FormatBGRA8888`). Distinguish canvases by **decoded pixel value**, never by dimensions.
- Canvas payload byte order is **B, G, R, A** — payload `{0x11,0x22,0x33,0xFF}` decodes to `color.NRGBA{R:0x33, G:0x22, B:0x11, A:0xFF}`.

---

### Task 1: Test scaffolding for WZ icon fixtures

Builds the shared fixture helpers every later test uses. Nothing is asserted about the fix yet — this task exists so Task 2's failing test is a one-liner and so the helpers get their own review gate.

**Files:**
- Create: `libs/atlas-wz/icons/fixture_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces, all in package `icons_test`:
  - `pixelPayload(t *testing.T, b, g, r, a byte) []byte` — zlib-compressed 4-byte BGRA canvas payload.
  - `openFixture(t *testing.T, b *wztest.Builder) *wz.File` — builds, writes to `t.TempDir()`, opens, registers cleanup.
  - `pixelAt(t *testing.T, img image.Image) color.NRGBA` — decodes pixel (0,0) as NRGBA.
  - Marker constants `markDefault`, `markStand`, `markLink`, `markTopLevel` (type `color.NRGBA`) and their matching payload triples.

- [ ] **Step 1: Create the fixture helper file**

Create `libs/atlas-wz/icons/fixture_test.go`:

```go
package icons_test

import (
	"bytes"
	"compress/zlib"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/crypto"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// Marker pixels. wztest.Canvas hardcodes 1x1 / format 2 (FormatBGRA8888),
// so fixture canvases are indistinguishable by dimension — every test tells
// them apart by decoded pixel value instead. Payload byte order is B,G,R,A.
var (
	markDefault  = color.NRGBA{R: 0x33, G: 0x22, B: 0x11, A: 0xFF}
	markStand    = color.NRGBA{R: 0x66, G: 0x55, B: 0x44, A: 0xFF}
	markLink     = color.NRGBA{R: 0x99, G: 0x88, B: 0x77, A: 0xFF}
	markTopLevel = color.NRGBA{R: 0xCC, G: 0xBB, B: 0xAA, A: 0xFF}
)

// payloadFor returns the canvas payload that decodes to the given marker.
func payloadFor(t *testing.T, m color.NRGBA) []byte {
	t.Helper()
	return pixelPayload(t, m.B, m.G, m.R, m.A)
}

// pixelPayload builds a canvas payload for a single BGRA pixel. canvas
// .Decompress tries zlib first (isZlibHeader wants 0x78 followed by
// 0x9C/0xDA/0x01/0x5E); zlib.NewWriter's default compression emits 0x78 0x9C.
func pixelPayload(t *testing.T, b, g, r, a byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write([]byte{b, g, r, a}); err != nil {
		t.Fatalf("zlib write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zlib close: %v", err)
	}
	return buf.Bytes()
}

// openFixture serializes the builder to a temp Npc.wz and opens it.
func openFixture(t *testing.T, b *wztest.Builder) *wz.File {
	t.Helper()
	data, err := b.Build()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "Npc.wz")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	f, err := wz.Open(logrus.StandardLogger(), path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// newArchive returns a builder preloaded with the settings wz.Open accepts
// for these fixtures (verified: "Detected version 83 (hash=1876)").
func newArchive() *wztest.Builder {
	return wztest.NewBuilder().
		SetVersion(83).
		SetEncryption(crypto.EncryptionNone)
}

// pixelAt decodes the single pixel of a 1x1 fixture canvas.
func pixelAt(t *testing.T, img image.Image) color.NRGBA {
	t.Helper()
	if img == nil {
		t.Fatalf("nil image")
	}
	return color.NRGBAModel.Convert(img.At(0, 0)).(color.NRGBA)
}
```

- [ ] **Step 2: Verify the scaffolding compiles and existing tests still pass**

Run: `cd libs/atlas-wz && go test ./icons/ -v`
Expected: PASS. `TestPublicSurfaceExists` and `TestNormalizeId` run; the new file contributes no tests yet. Go does not flag unused package-level vars or funcs, so this compiles cleanly.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-wz/icons/fixture_test.go
git commit -m "test(atlas-wz): add WZ icon fixture helpers for task-196"
```

---

### Task 2: Prefer `info/default` for NPCs

The behavior change. Adds the failing test, then the four-line finder that fixes it.

**Files:**
- Create: `libs/atlas-wz/icons/npc_default_test.go`
- Modify: `libs/atlas-wz/icons/extract.go` (`ExtractNpcIcon` at line 94; add `findNpcCanvas` next to `findStandCanvas` at line 332)

**Interfaces:**
- Consumes from Task 1: `newArchive`, `openFixture`, `payloadFor`, `pixelAt`, `markDefault`, `markStand`.
- Produces: `findNpcCanvas(props []property.Property) *property.CanvasProperty`, unexported, used only by `ExtractNpcIcon`.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-wz/icons/npc_default_test.go`:

```go
package icons_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestNpcPrefersInfoDefault locks in the task-196 fix. Npc.wz/1101000.img
// carries its real 129x86 likeness at info/default and a 1x60 placeholder at
// stand/0; the placeholder used to win.
func TestNpcPrefersInfoDefault(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1101000.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markDefault))),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1101000)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markDefault {
		t.Errorf("got %+v, want info/default marker %+v", got, markDefault)
	}
}

// TestNpcFallsBackToStand is the regression guard for the ~1211 NPCs that
// have no info/default at all.
func TestNpcFallsBackToStand(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1002000.img",
			wztest.Sub("info", wztest.Int("hideName", 1)),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1002000)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markStand {
		t.Errorf("got %+v, want stand marker %+v", got, markStand)
	}
}
```

- [ ] **Step 2: Run the tests to verify the first one fails**

Run: `cd libs/atlas-wz && go test ./icons/ -run 'TestNpcPrefersInfoDefault|TestNpcFallsBackToStand' -v`
Expected: `TestNpcPrefersInfoDefault` FAILS with `got {R:102 G:85 B:68 A:255}, want info/default marker {R:51 G:34 B:17 A:255}` — the stand marker, because `findStandCanvas` reaches `stand` first. `TestNpcFallsBackToStand` PASSES already.

- [ ] **Step 3: Add the NPC finder**

In `libs/atlas-wz/icons/extract.go`, immediately above `findStandCanvas`, add:

```go
// findNpcCanvas prefers info/default — the static likeness carried by NPCs
// whose stand animation is a 1-px placeholder (e.g. 1101000, 1101001) — then
// falls back to the shared stand/move ordering. Matching only a canvas among
// info's children excludes top-level `default` animation dirs such as
// 1209003.img's.
func findNpcCanvas(props []property.Property) *property.CanvasProperty {
	if info := findSub(props, "info"); info != nil {
		if cp := findSubCanvas(info.Children(), "default"); cp != nil {
			return cp
		}
	}
	return findStandCanvas(props)
}
```

- [ ] **Step 4: Point `ExtractNpcIcon` at the new finder**

In `libs/atlas-wz/icons/extract.go`, replace the body and doc comment of `ExtractNpcIcon`:

```go
// ExtractNpcIcon returns the decoded info/default (or stand/0 fallback)
// canvas for the given NPC id from a parsed Npc.wz file.
func ExtractNpcIcon(f *wz.File, id uint32) (image.Image, error) {
	return extractEntityIcon(f, id, findNpcCanvas)
}
```

Leave `ExtractMobIcon` and `ExtractReactorIcon` exactly as they are.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `cd libs/atlas-wz && go test ./icons/ -run 'TestNpcPrefersInfoDefault|TestNpcFallsBackToStand' -v`
Expected: both PASS.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-wz/icons/extract.go libs/atlas-wz/icons/npc_default_test.go
git commit -m "fix(atlas-wz): prefer NPC info/default canvas over placeholder stand/0

NPCs such as 1101000 and 1101001 ship a 1-px stand/0 placeholder and carry
their real likeness at info/default. findStandCanvas returned the
placeholder, so 33 NPCs in GMS 83.1 got a sliver icon.png. Give NPCs their
own finder; mobs and reactors keep findStandCanvas unchanged."
```

---

### Task 3: Guard the structural edge cases

Three tests that pin the boundaries the fix must not cross. No production code changes — if any of these fail, Task 2's implementation is wrong.

**Files:**
- Create: `libs/atlas-wz/icons/npc_default_edge_test.go`

**Interfaces:**
- Consumes from Task 1: `newArchive`, `openFixture`, `payloadFor`, `pixelAt`, `markDefault`, `markStand`, `markLink`, `markTopLevel`.
- Consumes from Task 2: the `findNpcCanvas` behavior, exercised through `icons.ExtractNpcIcon`.
- Produces: nothing consumed downstream.

- [ ] **Step 1: Write the edge-case tests**

Create `libs/atlas-wz/icons/npc_default_edge_test.go`:

```go
package icons_test

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-wz/icons"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestMobIgnoresInfoDefault proves the fix is NPC-scoped. Mob.wz contains no
// info/default nodes today, so a mob must keep resolving through stand/0 even
// when one is present.
func TestMobIgnoresInfoDefault(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("100100.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markDefault))),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
		)))

	img, err := icons.ExtractMobIcon(f, 100100)
	if err != nil {
		t.Fatalf("ExtractMobIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markStand {
		t.Errorf("got %+v, want stand marker %+v — mobs must not follow info/default", got, markStand)
	}
}

// TestNpcIgnoresTopLevelDefaultDir guards the 1209003.img shape: a top-level
// `default` imgdir holding a 14-frame animation, which is NOT info/default.
// That NPC has a healthy stand/0 and must keep it.
func TestNpcIgnoresTopLevelDefaultDir(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1209003.img",
			wztest.Sub("info", wztest.Int("hideName", 1)),
			wztest.Sub("stand", wztest.Canvas("0", payloadFor(t, markStand))),
			wztest.Sub("default", wztest.Canvas("0", payloadFor(t, markTopLevel))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1209003)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markStand {
		t.Errorf("got %+v, want stand marker %+v — a top-level default dir is not info/default", got, markStand)
	}
}

// TestNpcInfoDefaultBeatsLink covers the 2 NPCs carrying both info/default and
// info/link: the link is a fallback, not an override.
func TestNpcInfoDefaultBeatsLink(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1101010.img",
			wztest.Sub("info",
				wztest.Str("link", "1101011"),
				wztest.Canvas("default", payloadFor(t, markDefault)),
			),
		)).
		AddImage(wztest.Img("1101011.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markLink))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1101010)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markDefault {
		t.Errorf("got %+v, want own info/default %+v", got, markDefault)
	}
}

// TestNpcFollowsLinkToInfoDefault is the regression guard for the 33 linked
// NPCs that carry no canvas of their own. Verified working before the fix;
// must stay working after.
func TestNpcFollowsLinkToInfoDefault(t *testing.T) {
	f := openFixture(t, newArchive().
		AddImage(wztest.Img("1101020.img",
			wztest.Sub("info", wztest.Str("link", "1101021")),
		)).
		AddImage(wztest.Img("1101021.img",
			wztest.Sub("info", wztest.Canvas("default", payloadFor(t, markLink))),
		)))

	img, err := icons.ExtractNpcIcon(f, 1101020)
	if err != nil {
		t.Fatalf("ExtractNpcIcon: %v", err)
	}
	if got := pixelAt(t, img); got != markLink {
		t.Errorf("got %+v, want link-target marker %+v", got, markLink)
	}
}
```

- [ ] **Step 2: Run the edge tests**

Run: `cd libs/atlas-wz && go test ./icons/ -v`
Expected: all PASS, including the pre-existing `TestPublicSurfaceExists` and `TestNormalizeId`.

If `TestMobIgnoresInfoDefault` fails, `ExtractMobIcon` was changed — revert it to `findStandCanvas`. If `TestNpcIgnoresTopLevelDefaultDir` fails, `findNpcCanvas` is matching a sub-property instead of a canvas — confirm it uses `findSubCanvas`, not `findSub`.

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-wz/icons/npc_default_edge_test.go
git commit -m "test(atlas-wz): guard NPC info/default edge cases

Mob scoping, the 1209003 top-level `default` animation dir, and both
info/link directions."
```

---

### Task 4: Full verification sweep

Runs the `CLAUDE.md` gates across both affected modules and records the result.

**Files:**
- Modify: none (verification only)

**Interfaces:**
- Consumes: the completed implementation from Tasks 1–3.
- Produces: nothing.

- [ ] **Step 1: Test and vet `libs/atlas-wz` with the race detector**

Run:
```bash
cd libs/atlas-wz && go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean, exit 0.

- [ ] **Step 2: Test and vet `services/atlas-data`**

It consumes `libs/atlas-wz` through `go.work`, so it must be re-verified even though no file in it changed.

Run:
```bash
cd services/atlas-data/atlas.com/data && go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean, exit 0.

- [ ] **Step 3: Run the repo-root guards**

Run from the worktree root:
```bash
tools/redis-key-guard.sh && tools/goroutine-guard.sh
```
Expected: both exit 0. Neither is plausibly affected by this change, but `CLAUDE.md` requires them alongside `go vet`.

- [ ] **Step 4: Run the lint and format guard**

Run from the worktree root:
```bash
tools/lint.sh --check
```
Expected: exit 0. If it fails on formatting, run `tools/lint.sh` (no flags) to fix in place, then re-run `--check`.

Note: `tools/lint.sh --check` false-fails when nvm is not loaded, because the atlas-ui half cannot find node. If it fails on the frontend leg rather than on Go, load nvm 22 and re-run before treating it as a real failure.

- [ ] **Step 5: Confirm no `go.mod` was touched**

Run: `git diff --name-only main...HEAD -- '*go.mod' '*go.sum'`
Expected: empty output. If anything prints, `docker buildx bake atlas-data` becomes mandatory before the branch can be called done — run it and confirm it succeeds.

- [ ] **Step 6: Confirm the change surface is exactly what was planned**

Run: `git diff --stat main...HEAD`
Expected: `docs/tasks/task-196-npc-info-default-icon/` docs plus exactly four files under `libs/atlas-wz/icons/` — `extract.go`, `fixture_test.go`, `npc_default_test.go`, `npc_default_edge_test.go`. Nothing under `services/`.

- [ ] **Step 7: Commit any lint fixes**

Only if Step 4 rewrote files:

```bash
git add -A
git commit -m "chore(atlas-wz): apply lint fixes for task-196"
```

---

## Post-merge: the deploy step

**This is not optional and it is not covered by any test.** Icons live in the
assets bucket and are written only at WZ-ingest; `baseline/publish.go` tars the
database, not the assets. Merging this fixes nothing that is already deployed.

Each tenant needs `Npc.wz` re-ingested, per version, before the corrected icons
appear in atlas-ui. Expect 33 NPCs to gain a real likeness and 41 to change
from the full-body field sprite to the `info/default` portrait crop — the
latter is the accepted, visually-unverified cost recorded in `design.md`.

## Self-Review

Checked against `design.md`:

**Spec coverage.** Design "Change 1" → Task 2. "Change 2 — none" → enforced by
Task 4 Step 6, which asserts nothing under `services/` changed. All five
testing scenarios are covered: (1) the fix → `TestNpcPrefersInfoDefault`,
(2) no regression → `TestNpcFallsBackToStand`, (3) mobs unaffected →
`TestMobIgnoresInfoDefault`, (4) sub-dir exclusion →
`TestNpcIgnoresTopLevelDefaultDir`, (5) link precedence →
`TestNpcInfoDefaultBeatsLink`, plus `TestNpcFollowsLinkToInfoDefault` as an
extra guard on the direction that already worked. The design's "Verification"
section maps to Task 4. The rollout requirement is carried in "Post-merge".

**Placeholder scan.** No TBD/TODO, no "add error handling", no "similar to
Task N". Every code step carries complete, runnable code.

**Type consistency.** `findNpcCanvas` has the same
`func([]property.Property) *property.CanvasProperty` shape as `findStandCanvas`
and `findReactorCanvas`, so it satisfies the existing `canvasFinder` type used
by `extractEntityIcon`. Helper names are used identically in Tasks 2 and 3:
`newArchive`, `openFixture`, `payloadFor`, `pixelAt`. Marker vars
`markDefault`/`markStand`/`markLink`/`markTopLevel` are declared once in Task 1
and all four are consumed. Task 1's Interfaces block names `pixelPayload`
directly; `payloadFor` wraps it and is what Tasks 2 and 3 call — both are
defined in the Task 1 code.
