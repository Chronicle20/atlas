# Self-hosted character render service — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the runtime dependency on `maplestory.io` with an in-cluster atlas-wz-extractor render service, backed by atlas-assets nginx static caching and Character.wz extracts on `atlas-assets-pvc`.

**Architecture:** Three services cooperate. atlas-ui builds one URL `/api/assets/{tenant}/{region}/{v}/character/{hash}.png?<query>`. atlas-assets nginx `try_files` the cached PNG; on miss it `proxy_pass`es to atlas-wz-extractor's `/api/wz/character/render/{tenant}/{region}/{v}/{hash}.png`. atlas-wz-extractor's new `characterrender` HTTP handler validates the request, calls a new `characterimage` compositor that walks the joint tree using sidecar JSON metadata produced at extraction time, atomically writes the PNG to the shared PVC, and returns the bytes. Subsequent requests hit the file directly via nginx.

**Tech Stack:** Go (atlas-wz-extractor), `image/png` + `image/draw` (stdlib compositor), `crypto/sha256` (loadout hash), TypeScript + React (atlas-ui), `js-sha256`, TanStack React Query, nginx (atlas-assets).

---

## File structure

### Backend (`services/atlas-wz-extractor/atlas.com/wz-extractor/`)

| Path | Action | Responsibility |
|---|---|---|
| `image/extract.go` | modify | Extend `case name == "character":` to call new `extractCharacterParts`. Add new `case name == "base":` that calls new `extractCharacterMaps`. |
| `image/character_parts.go` | create | `extractCharacterParts(l, f, outputDir)` — walks Character.wz; emits per-`(templateId, stance, frame, partName).png` + `.json` sidecars and `{templateId}/info.json`. |
| `image/zmap.go` | create | `extractCharacterMaps(l, f, outputDir)` — emits `character-meta/zmap.json` and `smap.json` from `Base.wz`. |
| `extraction/processor.go` | modify | Wipe `{OUTPUT_IMG_DIR}/{tenant}/{region}/{v}/character/` before the WZ loop. |
| `characterimage/doc.go` | create | Package doc. |
| `characterimage/errors.go` | create | `ErrUnknownTemplateId`, `ErrInvalidStance`, `ErrFrameOutOfRange`, `ErrAssetsMissing`. |
| `characterimage/meta.go` | create | `LoadZmap`, `LoadSmap`, `LoadInfo`, `LoadPartMeta` — read JSONs from disk. |
| `characterimage/meta_cache.go` | create | In-process `*templateMeta` LRU keyed by templateId; `sync.Map` + `sync.Once` per key. |
| `characterimage/joints.go` | create | Joint-tree resolution: maps each part's `origin` to the canvas anchor via parent joint coordinates. |
| `characterimage/stance.go` | create | Stance/frame validation against the body skin's extracted directory. |
| `characterimage/compositor.go` | create | `Compositor.Composite(req CompositeRequest) (*image.RGBA, error)`. |
| `characterimage/scale.go` | create | Nearest-neighbor upscale `(96, 128) → (96r, 128r)`. |
| `characterimage/compositor_test.go` | create | Pixel-fixture tests for canonical loadouts. |
| `characterimage/joints_test.go` | create | Synthesised sidecars; assert canvas anchors. |
| `characterimage/stance_test.go` | create | Validation tests. |
| `characterrender/doc.go` | create | Package doc. |
| `characterrender/hash.go` | create | `CanonicalLoadoutString`, `LoadoutHash`. |
| `characterrender/hash_test.go` | create | Cross-language fixture parity. |
| `characterrender/path.go` | create | `ParseRenderPath(r *http.Request) (renderPath, error)`. |
| `characterrender/query.go` | create | `ParseRenderQuery(values url.Values) (renderQuery, error)`. |
| `characterrender/query_test.go` | create | Validation matrix. |
| `characterrender/error.go` | create | JSON:API-style error body writer. |
| `characterrender/write.go` | create | Atomic temp+rename PNG writer. |
| `characterrender/write_test.go` | create | Concurrent-write correctness. |
| `characterrender/otel.go` | create | `character.render` span helpers + Prometheus counters. |
| `characterrender/handler.go` | create | `HandleRender` — orchestrates parse → validate → composite → write → respond. |
| `characterrender/handler_test.go` | create | End-to-end via `httptest`. |
| `characterrender/resource.go` | create | `InitResource(h Handler)(si) server.RouteInitializer`. |
| `characterrender/testdata/loadout-hashes.json` | create | Cross-language fixture: rows of `{tenant, region, major, minor, skin, hair, face, stance, frame, resize, items, expectedHash, canonical}`. |
| `main.go` | modify | Add a second `AddRouteInitializer(characterrender.InitResource(...))` call. |

### Shared lib

| Path | Action | Responsibility |
|---|---|---|
| `libs/atlas-constants/item/constants.go` | modify | Add `IsTwoHanded(id Id) bool`. |
| `libs/atlas-constants/item/constants_test.go` | modify | Add tests for `IsTwoHanded`. |

### Deploy

| Path | Action | Responsibility |
|---|---|---|
| `services/atlas-assets/nginx.conf` | modify | Add `location ~ ^/(?<tenant>...)` block above `location /`. |

### Frontend (`services/atlas-ui/`)

| Path | Action | Responsibility |
|---|---|---|
| `package.json` | modify | Add `js-sha256` dependency. |
| `src/services/api/characterRender.service.ts` | create | URL builder, canonical string, hash, slot filter, character-to-loadout adapter. |
| `src/services/api/__tests__/characterRender.service.test.ts` | create | Hash parity (loads same testdata fixture), URL formation, slot dropping. |
| `src/services/api/maplestory.service.ts` | delete | After migration. |
| `src/services/api/__tests__/maplestory.service.test.ts` | delete | After migration. |
| `src/services/api/index.ts` | modify | Re-export from `characterRender.service.ts`; remove `mapleStoryService`. |
| `src/types/models/maplestory.ts` | modify | Keep types still used (`CharacterRenderOptions`, `MapleStoryCharacterData`); drop `WeaponType`, `SkinColorMapping`, `EquipmentSlotMapping` if unused after migration. |
| `src/lib/hooks/useCharacterImage.ts` | modify | Swap to new builder; queryKey extends with loadout hash. |
| `src/components/features/characters/CharacterRenderer.tsx` | modify | Replace `mapleStoryService` import with `characterRenderService`. |
| `src/components/features/characters/OptimizedCharacterRenderer.tsx` | modify | Same swap. |
| `src/components/features/characters/__tests__/CharacterRenderer.test.tsx` | modify | Update mock URLs to the new shape. |
| `src/lib/utils/character-cache-sw.ts` | modify | Replace hard-coded `maplestory.io` URL with the new builder. |
| `public/sw-character-cache.js` | modify | Update cached-URL allowlist if it references `maplestory.io`. |

---

## Implementation order

The plan is grouped into eight phases. Each phase is committable and testable before the next begins.

1. **Phase 1** — Shared `IsTwoHanded` helper.
2. **Phase 2** — Loadout hash + cross-language fixture (Go).
3. **Phase 3** — `Base.wz` zmap/smap extraction.
4. **Phase 4** — `Character.wz` worn-sprite extraction.
5. **Phase 5** — Compositor primitives.
6. **Phase 6** — Render handler + atomic write + route registration.
7. **Phase 7** — nginx `try_files` block + extraction-time cache wipe.
8. **Phase 8** — atlas-ui cutover (URL builder, hook, renderer, cleanup).

Phase 8 lands as a single PR; Phases 1–7 may stage on the backend behind an unused render route.

---

## Phase 1 — Shared `IsTwoHanded` helper

### Task 1.1: Add `item.IsTwoHanded`

**Files:**
- Modify: `libs/atlas-constants/item/constants.go`
- Modify: `libs/atlas-constants/item/constants_test.go`

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-constants/item/constants_test.go`:

```go
func TestIsTwoHanded(t *testing.T) {
	cases := []struct {
		name string
		id   Id
		want bool
	}{
		{"one-handed sword 130xxxx", Id(1302000), false},
		{"dagger 133xxxx", Id(1332000), false},
		{"wand 137xxxx", Id(1372000), false},
		{"two-handed sword 140xxxx", Id(1402000), true},
		{"polearm 144xxxx", Id(1442000), true},
		{"bow 145xxxx", Id(1452000), true},
		{"crossbow 146xxxx", Id(1462000), true},
		{"claw 147xxxx (one-handed)", Id(1472000), false},
		{"knuckle 148xxxx", Id(1482000), true},
		{"gun 149xxxx", Id(1492000), true},
		{"non-weapon hat 100xxxx", Id(1002000), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTwoHanded(tc.id); got != tc.want {
				t.Fatalf("IsTwoHanded(%d) = %v, want %v", tc.id, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/...`
Expected: FAIL with `undefined: IsTwoHanded`.

- [ ] **Step 3: Write the implementation**

Append to `libs/atlas-constants/item/constants.go`:

```go
// IsTwoHanded reports whether an equipped weapon templateId is a two-handed
// weapon. Two-handed weapons force the character renderer's stand2 stance.
// Returns false for non-weapon ids.
func IsTwoHanded(id Id) bool {
	switch GetWeaponType(id) {
	case WeaponTypeTwoHandedSword,
		WeaponTypeTwoHandedAxe,
		WeaponTypeTwoHandedMace,
		WeaponTypeSpear,
		WeaponTypePolearm,
		WeaponTypeBow,
		WeaponTypeCrossbow,
		WeaponTypeKnuckle,
		WeaponTypeGun:
		return true
	default:
		return false
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./item/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/item/constants.go libs/atlas-constants/item/constants_test.go
git commit -m "feat(atlas-constants): add item.IsTwoHanded helper"
```

---

## Phase 2 — Loadout hash + cross-language fixture

### Task 2.1: Create the cross-language fixture file

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/testdata/loadout-hashes.json`

- [ ] **Step 1: Compute expected hashes for fixture rows**

The fixture pins the canonical-string-to-hash mapping. Each row is independently verifiable: take the `canonical` field, SHA-256 it, hex-encode, take the first 16 chars.

Generate locally:

```bash
printf 'tenant-a|GMS|83.1|0|30030|20000|stand1|0|2|' | sha256sum | cut -c1-16
printf 'tenant-a|GMS|83.1|0|30030|20000|stand1|0|2|1002357' | sha256sum | cut -c1-16
printf 'tenant-a|GMS|83.1|2|30030|20000|stand2|0|2|1002357,1402024,1442024' | sha256sum | cut -c1-16
printf 'tenant-b|GMS|83.1|0|30030|20000|walk1|1|4|1002357' | sha256sum | cut -c1-16
printf 'tenant-a|JMS|83.1|0|30030|20000|stand1|0|1|' | sha256sum | cut -c1-16
```

Use whatever values the commands print as `expectedHash` below.

- [ ] **Step 2: Write the fixture**

```json
{
  "rows": [
    {
      "tenant": "tenant-a",
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "skin": 0,
      "hair": 30030,
      "face": 20000,
      "stance": "stand1",
      "frame": 0,
      "resize": 2,
      "items": [],
      "canonical": "tenant-a|GMS|83.1|0|30030|20000|stand1|0|2|",
      "expectedHash": "<hash from step 1, row 1>"
    },
    {
      "tenant": "tenant-a",
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "skin": 0,
      "hair": 30030,
      "face": 20000,
      "stance": "stand1",
      "frame": 0,
      "resize": 2,
      "items": [1002357],
      "canonical": "tenant-a|GMS|83.1|0|30030|20000|stand1|0|2|1002357",
      "expectedHash": "<hash from step 1, row 2>"
    },
    {
      "tenant": "tenant-a",
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "skin": 2,
      "hair": 30030,
      "face": 20000,
      "stance": "stand2",
      "frame": 0,
      "resize": 2,
      "items": [1002357, 1402024, 1442024],
      "canonical": "tenant-a|GMS|83.1|2|30030|20000|stand2|0|2|1002357,1402024,1442024",
      "expectedHash": "<hash from step 1, row 3>"
    },
    {
      "tenant": "tenant-b",
      "region": "GMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "skin": 0,
      "hair": 30030,
      "face": 20000,
      "stance": "walk1",
      "frame": 1,
      "resize": 4,
      "items": [1002357],
      "canonical": "tenant-b|GMS|83.1|0|30030|20000|walk1|1|4|1002357",
      "expectedHash": "<hash from step 1, row 4>"
    },
    {
      "tenant": "tenant-a",
      "region": "JMS",
      "majorVersion": 83,
      "minorVersion": 1,
      "skin": 0,
      "hair": 30030,
      "face": 20000,
      "stance": "stand1",
      "frame": 0,
      "resize": 1,
      "items": [],
      "canonical": "tenant-a|JMS|83.1|0|30030|20000|stand1|0|1|",
      "expectedHash": "<hash from step 1, row 5>"
    }
  ]
}
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/testdata/loadout-hashes.json
git commit -m "test(characterrender): add cross-language loadout hash fixture"
```

### Task 2.2: Write `CanonicalLoadoutString` (Go)

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/hash.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/hash_test.go`

- [ ] **Step 1: Write the failing test**

`hash_test.go`:

```go
package characterrender

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type hashFixtureRow struct {
	Tenant       string `json:"tenant"`
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
	Skin         int    `json:"skin"`
	Hair         int    `json:"hair"`
	Face         int    `json:"face"`
	Stance       string `json:"stance"`
	Frame        int    `json:"frame"`
	Resize       int    `json:"resize"`
	Items        []int  `json:"items"`
	Canonical    string `json:"canonical"`
	ExpectedHash string `json:"expectedHash"`
}

type hashFixture struct {
	Rows []hashFixtureRow `json:"rows"`
}

func loadHashFixture(t *testing.T) hashFixture {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "loadout-hashes.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var f hashFixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return f
}

func TestCanonicalLoadoutStringMatchesFixture(t *testing.T) {
	f := loadHashFixture(t)
	for _, row := range f.Rows {
		t.Run(row.Tenant+"-"+row.Stance, func(t *testing.T) {
			got := CanonicalLoadoutString(
				row.Tenant, row.Region, row.MajorVersion, row.MinorVersion,
				row.Skin, row.Hair, row.Face, row.Stance, row.Frame, row.Resize,
				row.Items,
			)
			if got != row.Canonical {
				t.Fatalf("canonical mismatch:\n got = %q\nwant = %q", got, row.Canonical)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterrender/...`
Expected: FAIL with `undefined: CanonicalLoadoutString`.

- [ ] **Step 3: Write the implementation**

`hash.go`:

```go
// Package characterrender exposes the HTTP handler that turns a character
// loadout into a deterministic PNG. Hash and canonical-string helpers are
// shared between the path-validation step and the cross-language fixture.
package characterrender

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CanonicalLoadoutString returns the canonical input string used to derive a
// loadout hash. Equipment ids are sorted ascending so that input order does
// not affect the hash.
func CanonicalLoadoutString(
	tenant, region string,
	majorVersion, minorVersion uint16,
	skin, hair, face int,
	stance string,
	frame, resize int,
	items []int,
) string {
	sorted := append([]int(nil), items...)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.Itoa(id)
	}
	return fmt.Sprintf(
		"%s|%s|%d.%d|%d|%d|%d|%s|%d|%d|%s",
		tenant, region, majorVersion, minorVersion,
		skin, hair, face, stance, frame, resize,
		strings.Join(parts, ","),
	)
}

// LoadoutHash returns the first 16 hex chars of SHA-256(canonical).
func LoadoutHash(canonical string) string {
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])[:16]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterrender/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/hash.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/hash_test.go
git commit -m "feat(characterrender): add CanonicalLoadoutString helper"
```

### Task 2.3: Verify `LoadoutHash` matches fixture

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/hash_test.go`

- [ ] **Step 1: Append the failing test**

Append to `hash_test.go`:

```go
func TestLoadoutHashMatchesFixture(t *testing.T) {
	f := loadHashFixture(t)
	for _, row := range f.Rows {
		t.Run(row.Tenant+"-"+row.Stance, func(t *testing.T) {
			canonical := CanonicalLoadoutString(
				row.Tenant, row.Region, row.MajorVersion, row.MinorVersion,
				row.Skin, row.Hair, row.Face, row.Stance, row.Frame, row.Resize,
				row.Items,
			)
			got := LoadoutHash(canonical)
			if got != row.ExpectedHash {
				t.Fatalf("hash mismatch for %s: got %s, want %s",
					row.Canonical, got, row.ExpectedHash)
			}
		})
	}
}

func TestCanonicalLoadoutStringSortsItems(t *testing.T) {
	a := CanonicalLoadoutString("t", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 2,
		[]int{1442024, 1002357, 1402024})
	b := CanonicalLoadoutString("t", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 2,
		[]int{1002357, 1402024, 1442024})
	if a != b {
		t.Fatalf("sort-invariance broken:\n a=%q\n b=%q", a, b)
	}
}
```

- [ ] **Step 2: Run tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterrender/...`
Expected: PASS — fixture rows now verify both canonical and hash.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/hash_test.go
git commit -m "test(characterrender): verify LoadoutHash against fixture"
```

---

## Phase 3 — `Base.wz` zmap/smap extraction

### Task 3.1: Define metadata sidecar JSON shapes

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/zmap.go`

- [ ] **Step 1: Write the file with type definitions and a stub function**

`image/zmap.go`:

```go
package image

import (
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// extractCharacterMaps reads zmap.img and smap.img from a Base.wz file and
// writes character-meta/zmap.json and character-meta/smap.json so the render
// service can resolve sprite z-order and slot-precedence at composition time.
func extractCharacterMaps(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}

	dir := filepath.Join(outputDir, "character-meta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create character-meta dir: %w", err)
	}

	if err := writeZmap(l, root.Images(), dir); err != nil {
		l.WithError(err).Warn("zmap extraction failed")
	}
	if err := writeSmap(l, root.Images(), dir); err != nil {
		l.WithError(err).Warn("smap extraction failed")
	}
	return nil
}

// writeZmap serializes Base.wz/zmap.img as an ordered list of layer-string
// names. The order in the WZ is the render order.
func writeZmap(l logrus.FieldLogger, images []*wz.Image, dir string) error {
	zmap := findImage(images, "zmap")
	if zmap == nil {
		return fmt.Errorf("zmap.img not found in Base.wz")
	}
	out := make([]string, 0, len(zmap.Properties()))
	for _, p := range zmap.Properties() {
		out = append(out, p.Name())
	}
	return writeJSON(filepath.Join(dir, "zmap.json"), out)
}

// writeSmap serializes Base.wz/smap.img as a layer-string -> slot-categories
// map (the WZ value is a string of slot codes, e.g. "CpHdH1H2H3...").
func writeSmap(l logrus.FieldLogger, images []*wz.Image, dir string) error {
	smap := findImage(images, "smap")
	if smap == nil {
		return fmt.Errorf("smap.img not found in Base.wz")
	}
	out := map[string]string{}
	for _, p := range smap.Properties() {
		if sp, ok := p.(*property.StringProperty); ok {
			out[sp.Name()] = sp.Value()
		}
	}
	return writeJSON(filepath.Join(dir, "smap.json"), out)
}

func findImage(images []*wz.Image, name string) *wz.Image {
	for _, img := range images {
		if strings.EqualFold(img.Name(), name) {
			return img
		}
	}
	return nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./image/...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/image/zmap.go
git commit -m "feat(wz-extractor): scaffold Base.wz zmap/smap extraction"
```

### Task 3.2: Wire `extractCharacterMaps` into the dispatch

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/extract.go`

- [ ] **Step 1: Add the new dispatch case**

In `image/extract.go`, change the switch in `ExtractIcons` to include a `base` case:

```go
	switch {
	case name == "npc":
		return extractEntityIcons(l, f, outputDir, "npc", findStandCanvas)
	case name == "mob":
		return extractEntityIcons(l, f, outputDir, "mob", findStandCanvas)
	case name == "reactor":
		return extractEntityIcons(l, f, outputDir, "reactor", findReactorCanvas)
	case name == "item":
		return extractItemIcons(l, f, outputDir)
	case name == "skill":
		return extractSkillIcons(l, f, outputDir)
	case name == "character":
		return extractEquipmentIcons(l, f, outputDir)
	case name == "ui":
		return extractUIIcons(l, f, outputDir)
	case name == "base":
		return extractCharacterMaps(l, f, outputDir)
	default:
		return nil
	}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./...`
Expected: success.

- [ ] **Step 3: Write a unit test using a synthetic image set**

Create `services/atlas-wz-extractor/atlas.com/wz-extractor/image/zmap_test.go`:

```go
package image

import (
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteZmapAndSmap(t *testing.T) {
	dir := t.TempDir()

	zmapImg := fakeImage("zmap", []property.Property{
		property.NewNull("body"),
		property.NewNull("arm"),
		property.NewNull("hairOverHead"),
		property.NewNull("weapon"),
	})
	smapImg := fakeImage("smap", []property.Property{
		property.NewString("capOverHair", "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe"),
		property.NewString("weaponOverArm", "WpAr"),
	})

	if err := writeZmap(t.Logf().(testLogger).asLogger(), []*fakeImageT{zmapImg}.toImages(), dir); err != nil {
		t.Fatalf("writeZmap: %v", err)
	}
	if err := writeSmap(t.Logf().(testLogger).asLogger(), []*fakeImageT{smapImg}.toImages(), dir); err != nil {
		t.Fatalf("writeSmap: %v", err)
	}

	var z []string
	mustReadJSON(t, filepath.Join(dir, "zmap.json"), &z)
	if got, want := z, []string{"body", "arm", "hairOverHead", "weapon"}; !equalSlices(got, want) {
		t.Fatalf("zmap = %v, want %v", got, want)
	}

	var s map[string]string
	mustReadJSON(t, filepath.Join(dir, "smap.json"), &s)
	if s["capOverHair"] != "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe" {
		t.Fatalf("smap[capOverHair] = %q", s["capOverHair"])
	}
}

func mustReadJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
```

This test references helpers (`fakeImage`, `*fakeImageT`, `testLogger`) that do not exist yet — they would force a refactor of the production code path to accept a property slice instead of `*wz.File`. Replace the test with the variant that calls the real `*wz.File` path indirectly (next step).

- [ ] **Step 4: Replace the unit test with one that exercises the helpers directly**

Overwrite `image/zmap_test.go` with a leaner test that bypasses `*wz.File` by calling small package-private helpers; refactor `writeZmap` / `writeSmap` to accept `[]property.Property` so they are testable. The refactor is two lines: add a thin wrapper and have `extractCharacterMaps` extract `properties` from the `*wz.Image`.

Replace the bodies of `writeZmap` and `writeSmap` in `image/zmap.go`:

```go
func writeZmap(l logrus.FieldLogger, images []*wz.Image, dir string) error {
	zmap := findImage(images, "zmap")
	if zmap == nil {
		return fmt.Errorf("zmap.img not found in Base.wz")
	}
	return writeZmapFromProps(zmap.Properties(), dir)
}

func writeZmapFromProps(props []property.Property, dir string) error {
	out := make([]string, 0, len(props))
	for _, p := range props {
		out = append(out, p.Name())
	}
	return writeJSON(filepath.Join(dir, "zmap.json"), out)
}

func writeSmap(l logrus.FieldLogger, images []*wz.Image, dir string) error {
	smap := findImage(images, "smap")
	if smap == nil {
		return fmt.Errorf("smap.img not found in Base.wz")
	}
	return writeSmapFromProps(smap.Properties(), dir)
}

func writeSmapFromProps(props []property.Property, dir string) error {
	out := map[string]string{}
	for _, p := range props {
		if sp, ok := p.(*property.StringProperty); ok {
			out[sp.Name()] = sp.Value()
		}
	}
	return writeJSON(filepath.Join(dir, "smap.json"), out)
}
```

Now overwrite `image/zmap_test.go`:

```go
package image

import (
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteZmapFromProps(t *testing.T) {
	dir := t.TempDir()
	props := []property.Property{
		property.NewNull("body"),
		property.NewNull("arm"),
		property.NewNull("hairOverHead"),
		property.NewNull("weapon"),
	}
	if err := writeZmapFromProps(props, dir); err != nil {
		t.Fatalf("writeZmapFromProps: %v", err)
	}
	var got []string
	readJSON(t, filepath.Join(dir, "zmap.json"), &got)
	want := []string{"body", "arm", "hairOverHead", "weapon"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("zmap = %v, want %v", got, want)
	}
}

func TestWriteSmapFromProps(t *testing.T) {
	dir := t.TempDir()
	props := []property.Property{
		property.NewString("capOverHair", "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe"),
		property.NewString("weaponOverArm", "WpAr"),
		property.NewNull("ignored-non-string"),
	}
	if err := writeSmapFromProps(props, dir); err != nil {
		t.Fatalf("writeSmapFromProps: %v", err)
	}
	var got map[string]string
	readJSON(t, filepath.Join(dir, "smap.json"), &got)
	want := map[string]string{
		"capOverHair":   "CpHdH1H2H3H4H5H6HsHfHbAfAyAsAe",
		"weaponOverArm": "WpAr",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("smap = %v, want %v", got, want)
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}
```

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./image/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/image/extract.go services/atlas-wz-extractor/atlas.com/wz-extractor/image/zmap.go services/atlas-wz-extractor/atlas.com/wz-extractor/image/zmap_test.go
git commit -m "feat(wz-extractor): extract Base.wz zmap/smap to JSON"
```

---

## Phase 4 — `Character.wz` worn-sprite extraction

### Task 4.1: Sprite metadata sidecar struct + JSON writer

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go`

- [ ] **Step 1: Write the file scaffold**

```go
package image

import (
	"atlas-wz-extractor/wz"
	"atlas-wz-extractor/wz/canvas"
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// partSidecar is the JSON sidecar emitted next to each part PNG.
type partSidecar struct {
	Origin  vec               `json:"origin"`
	Map     map[string]vec    `json:"map,omitempty"`
	Z       string            `json:"z,omitempty"`
	Group   string            `json:"group,omitempty"`
	Delay   int               `json:"delay,omitempty"`
	Face    int               `json:"face,omitempty"`
}

type vec struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// templateInfo is the per-img info.json block.
type templateInfo struct {
	Islot string `json:"islot,omitempty"`
	Vslot string `json:"vslot,omitempty"`
	Cash  int    `json:"cash"`
}

// stancesInScope is the explicit allow-list of stances we extract. Skipping
// fly/prone/swing/etc. keeps the on-disk footprint manageable.
var stancesInScope = map[string]struct{}{
	"stand1": {},
	"stand2": {},
	"walk1":  {},
	"alert":  {},
	"jump":   {},
}

// equipmentSubdirs are the Character.wz subdirectories whose .img files we
// extract worn sprites for. Body skin imgs live at the root, not in a subdir.
var equipmentSubdirs = []string{
	"Cap", "Coat", "Longcoat", "Pants", "Shoes", "Glove",
	"Cape", "Shield", "Weapon", "Hair", "Face", "Accessory",
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./image/...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go
git commit -m "feat(wz-extractor): scaffold Character.wz part extraction types"
```

### Task 4.2: Extract `info` block to `info.json`

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go`

- [ ] **Step 1: Append the helpers**

```go
// extractInfoBlock returns a templateInfo populated from the `info` sub of
// an equipment img. Missing fields default to zero values.
func extractInfoBlock(props []property.Property) templateInfo {
	info := findSub(props, "info")
	if info == nil {
		return templateInfo{}
	}
	out := templateInfo{}
	for _, p := range info.Children() {
		switch v := p.(type) {
		case *property.StringProperty:
			switch v.Name() {
			case "islot":
				out.Islot = v.Value()
			case "vslot":
				out.Vslot = v.Value()
			}
		case *property.IntProperty:
			if v.Name() == "cash" {
				out.Cash = int(v.Value())
			}
		case *property.ShortProperty:
			if v.Name() == "cash" {
				out.Cash = int(v.Value())
			}
		}
	}
	return out
}

// writeInfoJSON writes {dir}/info.json.
func writeInfoJSON(dir string, info templateInfo) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal info: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, "info.json"), b, 0o644)
}
```

- [ ] **Step 2: Write a test**

Create `services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts_test.go`:

```go
package image

import (
	"atlas-wz-extractor/wz/property"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractInfoBlock(t *testing.T) {
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewString("islot", "Cp"),
			property.NewString("vslot", "Cp"),
			property.NewInt("cash", 0),
		}),
	}
	got := extractInfoBlock(props)
	if got.Islot != "Cp" || got.Vslot != "Cp" || got.Cash != 0 {
		t.Fatalf("unexpected info: %+v", got)
	}
}

func TestWriteInfoJSON(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "1002357")
	if err := writeInfoJSON(target, templateInfo{Islot: "Cp", Vslot: "Cp", Cash: 0}); err != nil {
		t.Fatalf("write: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(target, "info.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var ti templateInfo
	if err := json.Unmarshal(b, &ti); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ti.Islot != "Cp" {
		t.Fatalf("islot = %q", ti.Islot)
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./image/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts_test.go
git commit -m "feat(wz-extractor): extract Character.wz info blocks to JSON"
```

### Task 4.3: Extract sprite + sidecar for one canvas

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts_test.go`

- [ ] **Step 1: Append the canvas-extraction helpers**

```go
// extractPartCanvas decodes a single part canvas, writes the PNG, and writes
// the JSON sidecar. The destination dir is created if missing.
func extractPartCanvas(l logrus.FieldLogger, f *wz.File, cp *property.CanvasProperty, dir, partName string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	data, err := f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		return fmt.Errorf("read canvas: %w", err)
	}
	img, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), f.CanvasEncryptionKey())
	if err != nil {
		return fmt.Errorf("decompress canvas: %w", err)
	}

	pngPath := filepath.Join(dir, partName+".png")
	out, err := os.Create(pngPath)
	if err != nil {
		return fmt.Errorf("create png: %w", err)
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		return fmt.Errorf("encode png: %w", err)
	}

	sidecar := buildPartSidecar(cp.Children())
	b, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sidecar: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, partName+".json"), b, 0o644)
}

// buildPartSidecar walks the children of a part canvas to produce the
// metadata sidecar. Children that are absent in the WZ stay zero-valued.
func buildPartSidecar(children []property.Property) partSidecar {
	out := partSidecar{Map: map[string]vec{}}
	for _, c := range children {
		switch v := c.(type) {
		case *property.VectorProperty:
			if v.Name() == "origin" {
				out.Origin = vec{X: int(v.X()), Y: int(v.Y())}
			}
		case *property.StringProperty:
			switch v.Name() {
			case "z":
				out.Z = v.Value()
			case "group":
				out.Group = v.Value()
			}
		case *property.IntProperty:
			if v.Name() == "delay" {
				out.Delay = int(v.Value())
			}
		case *property.ShortProperty:
			if v.Name() == "face" {
				out.Face = int(v.Value())
			}
		case *property.SubProperty:
			if v.Name() == "map" {
				for _, jp := range v.Children() {
					if jv, ok := jp.(*property.VectorProperty); ok {
						out.Map[jv.Name()] = vec{X: int(jv.X()), Y: int(jv.Y())}
					}
				}
			}
		}
	}
	if len(out.Map) == 0 {
		out.Map = nil
	}
	return out
}
```

- [ ] **Step 2: Append a sidecar-only unit test**

Append to `image/character_parts_test.go`:

```go
func TestBuildPartSidecar(t *testing.T) {
	children := []property.Property{
		property.NewVector("origin", 19, 32),
		property.NewString("z", "body"),
		property.NewString("group", "skin"),
		property.NewInt("delay", 180),
		property.NewShort("face", 1),
		property.NewSub("map", []property.Property{
			property.NewVector("neck", -4, -32),
			property.NewVector("navel", -6, -20),
		}),
	}
	got := buildPartSidecar(children)
	if got.Origin != (vec{X: 19, Y: 32}) {
		t.Fatalf("origin = %+v", got.Origin)
	}
	if got.Z != "body" || got.Group != "skin" || got.Delay != 180 || got.Face != 1 {
		t.Fatalf("scalar mismatch: %+v", got)
	}
	if got.Map["neck"] != (vec{X: -4, Y: -32}) {
		t.Fatalf("map.neck = %+v", got.Map["neck"])
	}
	if got.Map["navel"] != (vec{X: -6, Y: -20}) {
		t.Fatalf("map.navel = %+v", got.Map["navel"])
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./image/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts_test.go
git commit -m "feat(wz-extractor): build Character.wz part sidecar"
```

### Task 4.4: Walk one img's stance/frame/part tree

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go`

- [ ] **Step 1: Append the walker**

```go
// extractTemplateImg processes one Character.wz .img file. It writes
// {outRoot}/{templateId}/info.json plus, for every supported stance/frame
// canvas, {outRoot}/{templateId}/{stance}/{frame}/{part}.png + .json.
func extractTemplateImg(l logrus.FieldLogger, f *wz.File, img *wz.Image, outRoot string) (int, error) {
	templateId := normalizeId(img.Name())
	templateDir := filepath.Join(outRoot, templateId)

	info := extractInfoBlock(img.Properties())
	if err := writeInfoJSON(templateDir, info); err != nil {
		return 0, fmt.Errorf("write info: %w", err)
	}

	count := 0
	for _, p := range img.Properties() {
		stanceSub, ok := p.(*property.SubProperty)
		if !ok {
			continue
		}
		stance := stanceSub.Name()
		if _, ok := stancesInScope[stance]; !ok {
			continue
		}
		for _, fp := range stanceSub.Children() {
			frameSub, ok := fp.(*property.SubProperty)
			if !ok {
				continue
			}
			frameName := frameSub.Name()
			frameDir := filepath.Join(templateDir, stance, frameName)
			for _, partProp := range frameSub.Children() {
				cp, ok := partProp.(*property.CanvasProperty)
				if !ok {
					continue
				}
				if err := extractPartCanvas(l, f, cp, frameDir, cp.Name()); err != nil {
					l.WithError(err).Warnf("extract part %s/%s/%s/%s", templateId, stance, frameName, cp.Name())
					continue
				}
				count++
			}
		}
	}
	return count, nil
}

// extractCharacterParts walks Character.wz: every .img at the root (body
// skins) plus every .img in equipmentSubdirs, emitting per-template assets.
func extractCharacterParts(l logrus.FieldLogger, f *wz.File, outputDir string) error {
	root := f.Root()
	if root == nil {
		return nil
	}
	tenantOut := filepath.Join(outputDir, "character-parts")
	total := 0

	for _, img := range root.Images() {
		// Only body skin imgs live at the root; their names start with "0000" or "0001".
		if !strings.HasPrefix(img.Name(), "0000") && !strings.HasPrefix(img.Name(), "0001") {
			continue
		}
		n, err := extractTemplateImg(l, f, img, tenantOut)
		if err != nil {
			l.WithError(err).Warnf("extract body skin %s", img.Name())
			continue
		}
		total += n
	}
	for _, sub := range equipmentSubdirs {
		dir := findCharSubdir(root.Directories(), sub)
		if dir == nil {
			continue
		}
		for _, img := range dir.Images() {
			n, err := extractTemplateImg(l, f, img, tenantOut)
			if err != nil {
				l.WithError(err).Warnf("extract %s/%s", sub, img.Name())
				continue
			}
			total += n
		}
	}
	l.Infof("Extracted [%d] character part canvases.", total)
	return nil
}

func findCharSubdir(dirs []*wz.Directory, name string) *wz.Directory {
	for _, d := range dirs {
		if strings.EqualFold(d.Name(), name) {
			return d
		}
	}
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./...`
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/image/character_parts.go
git commit -m "feat(wz-extractor): walk Character.wz stance/frame canvases"
```

### Task 4.5: Wire `extractCharacterParts` into the dispatch

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/image/extract.go`

- [ ] **Step 1: Combine icons + parts under the character case**

Replace the `case name == "character":` branch:

```go
	case name == "character":
		if err := extractEquipmentIcons(l, f, outputDir); err != nil {
			l.WithError(err).Warn("equipment icons extraction failed")
		}
		return extractCharacterParts(l, f, outputDir)
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./...`
Expected: success.

- [ ] **Step 3: Run all extractor tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./...`
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/image/extract.go
git commit -m "feat(wz-extractor): dispatch Character.wz to part extraction"
```

---

## Phase 5 — Compositor primitives

### Task 5.1: `characterimage` package skeleton + errors

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/doc.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/errors.go`

- [ ] **Step 1: Write doc.go**

```go
// Package characterimage composes character PNGs from Character.wz extracts.
// It is the only place in atlas-wz-extractor that knows about joint trees,
// z-order resolution, and slot precedence; the HTTP layer is in
// atlas-wz-extractor/characterrender.
package characterimage
```

- [ ] **Step 2: Write errors.go**

```go
package characterimage

import "errors"

var (
	ErrUnknownTemplateId = errors.New("characterimage: unknown templateId")
	ErrInvalidStance     = errors.New("characterimage: invalid stance")
	ErrFrameOutOfRange   = errors.New("characterimage: frame out of range")
	ErrAssetsMissing     = errors.New("characterimage: assets missing")
)
```

- [ ] **Step 3: Verify it compiles**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./characterimage/...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/
git commit -m "feat(characterimage): scaffold package + sentinel errors"
```

### Task 5.2: Sidecar loaders + meta cache

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/meta.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/meta_cache.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/meta_test.go`

- [ ] **Step 1: Write the loaders**

`meta.go`:

```go
package characterimage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Vec is a 2D coordinate pair used by both origin and joint maps.
type Vec struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// PartMeta is the JSON sidecar shape the extractor emits next to each PNG.
type PartMeta struct {
	Origin Vec            `json:"origin"`
	Map    map[string]Vec `json:"map"`
	Z      string         `json:"z"`
	Group  string         `json:"group"`
	Delay  int            `json:"delay"`
	Face   int            `json:"face"`
}

// TemplateInfo mirrors {templateId}/info.json on disk.
type TemplateInfo struct {
	Islot string `json:"islot"`
	Vslot string `json:"vslot"`
	Cash  int    `json:"cash"`
}

// LoadZmap reads character-meta/zmap.json. The slice is the render order.
func LoadZmap(assetsRoot string) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(assetsRoot, "character-meta", "zmap.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: zmap: %v", ErrAssetsMissing, err)
	}
	var out []string
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse zmap: %w", err)
	}
	return out, nil
}

// LoadSmap reads character-meta/smap.json: layer-string -> slot codes.
func LoadSmap(assetsRoot string) (map[string]string, error) {
	b, err := os.ReadFile(filepath.Join(assetsRoot, "character-meta", "smap.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: smap: %v", ErrAssetsMissing, err)
	}
	out := map[string]string{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("parse smap: %w", err)
	}
	return out, nil
}

// LoadInfo reads character-parts/{templateId}/info.json.
func LoadInfo(assetsRoot, templateId string) (TemplateInfo, error) {
	b, err := os.ReadFile(filepath.Join(assetsRoot, "character-parts", templateId, "info.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return TemplateInfo{}, fmt.Errorf("%w: %s", ErrUnknownTemplateId, templateId)
		}
		return TemplateInfo{}, fmt.Errorf("read info %s: %w", templateId, err)
	}
	var ti TemplateInfo
	if err := json.Unmarshal(b, &ti); err != nil {
		return TemplateInfo{}, fmt.Errorf("parse info %s: %w", templateId, err)
	}
	return ti, nil
}

// LoadPartMeta reads {templateId}/{stance}/{frame}/{part}.json.
func LoadPartMeta(assetsRoot, templateId, stance string, frame int, part string) (PartMeta, error) {
	path := filepath.Join(
		assetsRoot, "character-parts", templateId,
		stance, fmt.Sprintf("%d", frame), part+".json",
	)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PartMeta{}, fmt.Errorf("%w: %s", ErrAssetsMissing, path)
		}
		return PartMeta{}, fmt.Errorf("read part meta %s: %w", path, err)
	}
	var pm PartMeta
	if err := json.Unmarshal(b, &pm); err != nil {
		return PartMeta{}, fmt.Errorf("parse part meta %s: %w", path, err)
	}
	return pm, nil
}
```

`meta_cache.go`:

```go
package characterimage

import "sync"

// metaCache memoizes per-templateId TemplateInfo across renders within one
// process. Entries are never evicted (Character.wz is small enough; bounded
// by extraction wipe). Thread-safe via sync.Map.
type metaCache struct {
	infos sync.Map // templateId -> TemplateInfo
}

func newMetaCache() *metaCache { return &metaCache{} }

func (c *metaCache) info(assetsRoot, templateId string) (TemplateInfo, error) {
	if v, ok := c.infos.Load(templateId); ok {
		return v.(TemplateInfo), nil
	}
	ti, err := LoadInfo(assetsRoot, templateId)
	if err != nil {
		return TemplateInfo{}, err
	}
	c.infos.Store(templateId, ti)
	return ti, nil
}
```

- [ ] **Step 2: Write a smoke test**

`meta_test.go`:

```go
package characterimage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInfoUnknownTemplateId(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadInfo(dir, "9999999")
	if !errors.Is(err, ErrUnknownTemplateId) {
		t.Fatalf("expected ErrUnknownTemplateId, got %v", err)
	}
}

func TestLoadInfoRoundTrip(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "character-parts", "1002357")
	if err := os.MkdirAll(tmpl, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpl, "info.json"),
		[]byte(`{"islot":"Cp","vslot":"Cp","cash":0}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	ti, err := LoadInfo(dir, "1002357")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if ti.Islot != "Cp" || ti.Vslot != "Cp" {
		t.Fatalf("got %+v", ti)
	}
}

func TestMetaCacheMemoizes(t *testing.T) {
	dir := t.TempDir()
	tmpl := filepath.Join(dir, "character-parts", "1002357")
	_ = os.MkdirAll(tmpl, 0o755)
	_ = os.WriteFile(filepath.Join(tmpl, "info.json"),
		[]byte(`{"islot":"Cp","vslot":"Cp","cash":0}`), 0o644)

	c := newMetaCache()
	a, _ := c.info(dir, "1002357")
	// Delete the file; cached value must still be returned.
	_ = os.RemoveAll(tmpl)
	b, err := c.info(dir, "1002357")
	if err != nil {
		t.Fatalf("second call errored: %v", err)
	}
	if a != b {
		t.Fatalf("cache miss: %+v vs %+v", a, b)
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/meta.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/meta_cache.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/meta_test.go
git commit -m "feat(characterimage): add metadata loaders + cache"
```

### Task 5.3: Joint resolution

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/joints.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/joints_test.go`

- [ ] **Step 1: Write the failing test**

`joints_test.go`:

```go
package characterimage

import "testing"

// Body has its own neck joint at (-4,-32) relative to its origin.
// Hair declares origin (10, 50) and a complementary neck joint at (5, 12) on
// itself. When attached to body anchored at canvas (48, 96), the hair anchor
// must be placed so hair.neck on canvas == body.neck on canvas.
func TestResolveAnchorJoinsByJointName(t *testing.T) {
	body := PartMeta{Origin: Vec{X: 19, Y: 32}, Map: map[string]Vec{"neck": {X: -4, Y: -32}}}
	bodyAnchor := Anchor{X: 48, Y: 96}

	hair := PartMeta{Origin: Vec{X: 10, Y: 50}, Map: map[string]Vec{"neck": {X: 5, Y: 12}}}

	got := ResolveAnchor(bodyAnchor, body, hair, "neck")

	// body.neck on canvas = bodyAnchor + body.map.neck
	//                     = (48,96) + (-4,-32) = (44, 64)
	// hair.origin on canvas must be at (body.neck.canvas - hair.map.neck)
	//                                  = (44 - 5, 64 - 12) = (39, 52)
	if got != (Anchor{X: 39, Y: 52}) {
		t.Fatalf("ResolveAnchor = %+v, want {39 52}", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: FAIL with `undefined: Anchor`, `undefined: ResolveAnchor`.

- [ ] **Step 3: Write the implementation**

`joints.go`:

```go
package characterimage

import "fmt"

// Anchor is a canvas-space coordinate at which a part's `origin` is placed.
type Anchor struct{ X, Y int }

// ResolveAnchor computes the canvas anchor for a child part attached to a
// parent part via joint name. The child's `Map[joint]` describes the
// complementary point on the child sprite that should align with the
// parent's joint coordinate.
//
//	parentJointCanvas = parentAnchor + parent.Map[joint]
//	childAnchor       = parentJointCanvas - child.Map[joint]
//
// The child's `Origin` lands at `childAnchor`.
func ResolveAnchor(parentAnchor Anchor, parent, child PartMeta, joint string) Anchor {
	pj := parent.Map[joint]
	cj := child.Map[joint]
	return Anchor{
		X: parentAnchor.X + pj.X - cj.X,
		Y: parentAnchor.Y + pj.Y - cj.Y,
	}
}

// MustHaveJoint returns an error if either part lacks `joint`.
func MustHaveJoint(parent, child PartMeta, joint string) error {
	if _, ok := parent.Map[joint]; !ok {
		return fmt.Errorf("parent missing joint %q", joint)
	}
	if _, ok := child.Map[joint]; !ok {
		return fmt.Errorf("child missing joint %q", joint)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/joints.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/joints_test.go
git commit -m "feat(characterimage): joint-tree anchor resolution"
```

### Task 5.4: Stance / frame validation

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/stance.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/stance_test.go`

- [ ] **Step 1: Write the failing test**

`stance_test.go`:

```go
package characterimage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateStanceUnknown(t *testing.T) {
	if err := ValidateStance("warp"); !errors.Is(err, ErrInvalidStance) {
		t.Fatalf("got %v, want ErrInvalidStance", err)
	}
}

func TestValidateStanceKnown(t *testing.T) {
	for _, s := range []string{"stand1", "stand2", "walk1", "alert", "jump"} {
		if err := ValidateStance(s); err != nil {
			t.Fatalf("ValidateStance(%q) = %v", s, err)
		}
	}
}

func TestValidateFrameOutOfRange(t *testing.T) {
	dir := t.TempDir()
	frameDir := filepath.Join(dir, "character-parts", "00002000", "stand1", "0")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Only frame 0 exists. Asking for frame 1 must fail.
	if err := ValidateFrame(dir, "00002000", "stand1", 1); !errors.Is(err, ErrFrameOutOfRange) {
		t.Fatalf("got %v, want ErrFrameOutOfRange", err)
	}
	if err := ValidateFrame(dir, "00002000", "stand1", 0); err != nil {
		t.Fatalf("ValidateFrame frame 0: %v", err)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: FAIL — undefined `ValidateStance`/`ValidateFrame`.

- [ ] **Step 3: Write the implementation**

`stance.go`:

```go
package characterimage

import (
	"fmt"
	"os"
	"path/filepath"
)

var supportedStances = map[string]struct{}{
	"stand1": {}, "stand2": {}, "walk1": {}, "alert": {}, "jump": {},
}

// SupportedStances is the canonical list returned in 400 error meta.
func SupportedStances() []string {
	return []string{"stand1", "stand2", "walk1", "alert", "jump"}
}

// ValidateStance returns ErrInvalidStance if `s` is not in scope.
func ValidateStance(s string) error {
	if _, ok := supportedStances[s]; ok {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrInvalidStance, s)
}

// ValidateFrame checks whether {templateId}/{stance}/{frame} exists in the
// extract. Used against the body skin img only (other parts inherit).
func ValidateFrame(assetsRoot, templateId, stance string, frame int) error {
	path := filepath.Join(assetsRoot, "character-parts", templateId, stance, fmt.Sprintf("%d", frame))
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s/%s/%d", ErrFrameOutOfRange, templateId, stance, frame)
		}
		return fmt.Errorf("stat frame: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/stance.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/stance_test.go
git commit -m "feat(characterimage): stance and frame validation"
```

### Task 5.5: Skin mapping + slot filtering

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/skin.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/skin_test.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/filter.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/filter_test.go`

- [ ] **Step 1: Write skin mapping**

`skin.go`:

```go
package characterimage

import "fmt"

// internalSkinToWZ maps the atlas-ui internal 0..10 to the Character.wz id
// 2000..2013 (non-contiguous range). Source of truth lives in this file.
var internalSkinToWZ = map[int]int{
	0:  2000,
	1:  2001,
	2:  2002,
	3:  2003,
	4:  2004,
	5:  2005,
	6:  2009,
	7:  2010,
	8:  2011,
	9:  2012,
	10: 2013,
}

// MapInternalSkin returns the WZ skin id for an internal 0..10 value.
func MapInternalSkin(internal int) (int, error) {
	if wz, ok := internalSkinToWZ[internal]; ok {
		return wz, nil
	}
	return 0, fmt.Errorf("internal skin %d out of range 0..10", internal)
}
```

`skin_test.go`:

```go
package characterimage

import "testing"

func TestMapInternalSkin(t *testing.T) {
	cases := map[int]int{0: 2000, 5: 2005, 6: 2009, 10: 2013}
	for in, want := range cases {
		got, err := MapInternalSkin(in)
		if err != nil {
			t.Fatalf("MapInternalSkin(%d) errored: %v", in, err)
		}
		if got != want {
			t.Fatalf("MapInternalSkin(%d) = %d, want %d", in, got, want)
		}
	}
	if _, err := MapInternalSkin(11); err == nil {
		t.Fatalf("expected error for skin 11")
	}
}
```

- [ ] **Step 2: Write slot filter**

`filter.go`:

```go
package characterimage

// Slot keys that are silently dropped from a render request before
// compositing. Pet/mount/cash slots are intentionally not rendered in v1.
var droppedSlots = map[int]struct{}{
	-14: {}, // pet
	-18: {}, -19: {}, -20: {}, // mount
	-21: {}, -22: {}, -23: {}, -24: {}, -25: {},
	-26: {}, -27: {}, -28: {}, -29: {}, -30: {},
}

// FilterEquipment returns a copy of `in` with mount/pet/cash slots removed.
// Cash slots (-101..-114) are dropped via numeric range so we don't have to
// enumerate them.
func FilterEquipment(in map[int]int) map[int]int {
	out := make(map[int]int, len(in))
	for slot, id := range in {
		if _, dropped := droppedSlots[slot]; dropped {
			continue
		}
		if slot <= -101 && slot >= -114 {
			continue
		}
		out[slot] = id
	}
	return out
}
```

`filter_test.go`:

```go
package characterimage

import "testing"

func TestFilterEquipmentDropsMountPetCash(t *testing.T) {
	in := map[int]int{
		-1:   1002357, // hat
		-11:  1402024, // weapon
		-14:  5000000, // pet — drop
		-18:  1932000, // mount saddle — drop
		-19:  1932001, // mount — drop
		-21:  1012000, // pet ring slot — drop
		-101: 1002001, // cash hat — drop
		-114: 1132001, // cash belt — drop
	}
	out := FilterEquipment(in)
	if _, ok := out[-1]; !ok {
		t.Fatal("hat dropped")
	}
	if _, ok := out[-11]; !ok {
		t.Fatal("weapon dropped")
	}
	for _, slot := range []int{-14, -18, -19, -21, -101, -114} {
		if _, ok := out[slot]; ok {
			t.Fatalf("slot %d not dropped", slot)
		}
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/skin.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/skin_test.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/filter.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/filter_test.go
git commit -m "feat(characterimage): skin mapping + equipment slot filter"
```

### Task 5.6: Two-handed override

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/two_handed.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/two_handed_test.go`

- [ ] **Step 1: Write the failing test**

`two_handed_test.go`:

```go
package characterimage

import "testing"

func TestResolveStanceForcesStand2OnTwoHanded(t *testing.T) {
	// Polearm 1442024 is two-handed. Sword 1302000 is one-handed.
	got, override := ResolveStance("stand1", map[int]int{-11: 1442024})
	if got != "stand2" || !override {
		t.Fatalf("polearm forces stand2: got %q override=%v", got, override)
	}
	got, override = ResolveStance("stand1", map[int]int{-11: 1302000})
	if got != "stand1" || override {
		t.Fatalf("sword keeps stand1: got %q override=%v", got, override)
	}
	// walk1 must also be overridden when two-handed weapon equipped.
	got, override = ResolveStance("walk1", map[int]int{-11: 1442024})
	if got != "stand2" || !override {
		t.Fatalf("polearm + walk1: got %q override=%v", got, override)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: FAIL — undefined `ResolveStance`.

- [ ] **Step 3: Implementation**

`two_handed.go`:

```go
package characterimage

import "github.com/Chronicle20/atlas/libs/atlas-constants/item"

// ResolveStance applies the two-handed override: if any equipped weapon (slot
// -11) is two-handed, the rendered stance becomes "stand2" regardless of the
// requested value. Returns the override flag for observability.
func ResolveStance(requested string, equipment map[int]int) (string, bool) {
	weaponId, ok := equipment[-11]
	if !ok {
		return requested, false
	}
	if !item.IsTwoHanded(item.Id(weaponId)) {
		return requested, false
	}
	if requested == "stand2" {
		return "stand2", false
	}
	return "stand2", true
}
```

- [ ] **Step 4: Add the lib dep to wz-extractor go.mod**

If `atlas-constants` is not yet imported by `atlas-wz-extractor`, add the require:

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor
go get github.com/Chronicle20/atlas/libs/atlas-constants
```

This may already be present via the workspace; if `go test` succeeds without `go get`, skip this step.

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/two_handed.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/two_handed_test.go services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod services/atlas-wz-extractor/atlas.com/wz-extractor/go.sum
git commit -m "feat(characterimage): two-handed weapon stance override"
```

### Task 5.7: Nearest-neighbor scaling

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/scale.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/scale_test.go`

- [ ] **Step 1: Failing test**

`scale_test.go`:

```go
package characterimage

import (
	"image"
	"image/color"
	"testing"
)

func TestNearestNeighborUpscale2x(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(1, 0, color.RGBA{B: 255, A: 255})

	got := NearestNeighbor(src, 2)
	if got.Bounds().Dx() != 4 || got.Bounds().Dy() != 2 {
		t.Fatalf("dims = %v", got.Bounds())
	}
	if got.RGBAAt(0, 0).R != 255 {
		t.Fatalf("upper-left pixel not red")
	}
	if got.RGBAAt(1, 0).R != 255 {
		t.Fatalf("(1,0) should still be red (block expand)")
	}
	if got.RGBAAt(2, 0).B != 255 {
		t.Fatalf("(2,0) should be blue (block of source x=1)")
	}
	if got.RGBAAt(3, 1).B != 255 {
		t.Fatalf("(3,1) should be blue (block of source x=1)")
	}
}

func TestNearestNeighborScale1Identity(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 3))
	got := NearestNeighbor(src, 1)
	if got != src {
		t.Fatal("scale=1 should return the same image")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./characterimage/...`
Expected: FAIL — undefined `NearestNeighbor`.

- [ ] **Step 3: Implementation**

`scale.go`:

```go
package characterimage

import "image"

// NearestNeighbor upscales `src` by integer factor `scale`. Returns `src`
// unchanged when scale == 1. Pixel-art preserving — no smoothing.
func NearestNeighbor(src *image.RGBA, scale int) *image.RGBA {
	if scale <= 1 {
		return src
	}
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.RGBAAt(x, y)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					dst.SetRGBA(x*scale+dx, y*scale+dy, c)
				}
			}
		}
	}
	return dst
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/scale.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/scale_test.go
git commit -m "feat(characterimage): nearest-neighbor upscale"
```

### Task 5.8: Compositor — request type + bare-body render

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor_test.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/testdata/.gitkeep`

- [ ] **Step 1: Define types and a stub**

`compositor.go`:

```go
package characterimage

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CanvasWidth and CanvasHeight are the native, pre-resize compositing canvas
// dimensions. The body origin lands at (CanvasWidth/2, FootRow - 4).
const (
	CanvasWidth  = 96
	CanvasHeight = 128
	FootRow      = 124
)

// CompositeRequest describes one render. All slot filtering and skin mapping
// is done by the compositor — callers pass the raw request shape from the
// HTTP layer.
type CompositeRequest struct {
	AssetsRoot string         // absolute path: {OUTPUT_IMG_DIR}/{tenant}/{region}/{v}
	Skin       int            // internal 0..10
	Hair       int            // hair templateId
	Face       int            // face templateId
	Equipment  map[int]int    // raw slot -> templateId
	Stance     string         // requested stance; may be overridden
	Frame      int
	Resize     int            // 1..4
	IsMale     bool           // if true, use 0001{wzSkin}.img; else 0000{wzSkin}.img
}

// Compositor holds the per-process zmap/smap and meta cache.
type Compositor struct {
	zmap  []string
	smap  map[string]string
	cache *metaCache
}

// NewCompositor lazily loads zmap/smap from disk on first use.
func NewCompositor() *Compositor {
	return &Compositor{cache: newMetaCache()}
}

func (c *Compositor) loadMaps(assetsRoot string) error {
	if c.zmap == nil {
		z, err := LoadZmap(assetsRoot)
		if err != nil {
			return err
		}
		c.zmap = z
	}
	if c.smap == nil {
		s, err := LoadSmap(assetsRoot)
		if err != nil {
			return err
		}
		c.smap = s
	}
	return nil
}

// CompositeResult bundles the composited image and observability metadata.
type CompositeResult struct {
	Image              *image.RGBA
	ResolvedStance     string
	TwoHandedOverride  bool
	EquippedSlotCount  int
}

// Composite runs the algorithm:
//  1. filter equipment, 2. resolve stance, 3. map skin, 4. validate stance/frame,
//  5. blit body skin, 6. blit equipment by zmap order, 7. scale.
func (c *Compositor) Composite(req CompositeRequest) (*CompositeResult, error) {
	if err := ValidateStance(req.Stance); err != nil {
		return nil, err
	}
	if req.Resize < 1 || req.Resize > 4 {
		return nil, fmt.Errorf("resize out of range 1..4: %d", req.Resize)
	}
	if err := c.loadMaps(req.AssetsRoot); err != nil {
		return nil, err
	}

	filtered := FilterEquipment(req.Equipment)
	stance, override := ResolveStance(req.Stance, filtered)

	wzSkin, err := MapInternalSkin(req.Skin)
	if err != nil {
		return nil, err
	}
	bodyTemplate := bodyTemplateId(req.IsMale, wzSkin)

	if err := ValidateFrame(req.AssetsRoot, bodyTemplate, stance, req.Frame); err != nil {
		return nil, err
	}

	canvas := image.NewRGBA(image.Rect(0, 0, CanvasWidth, CanvasHeight))
	if err := c.blitBody(canvas, req.AssetsRoot, bodyTemplate, stance, req.Frame); err != nil {
		return nil, err
	}
	// Equipment blitting comes in Task 5.9.

	out := NearestNeighbor(canvas, req.Resize)
	return &CompositeResult{
		Image:             out,
		ResolvedStance:    stance,
		TwoHandedOverride: override,
		EquippedSlotCount: len(filtered),
	}, nil
}

// bodyTemplateId returns the WZ img name for a given gender + skin id.
// Female: 0000{skin}, male: 0001{skin}.
func bodyTemplateId(isMale bool, wzSkin int) string {
	prefix := "0000"
	if isMale {
		prefix = "0001"
	}
	return fmt.Sprintf("%s%d", prefix, wzSkin)
}

// blitBody anchors the body's `body` part at the canvas center and draws
// every part canvas in the body img's frame in zmap order.
func (c *Compositor) blitBody(canvas *image.RGBA, assetsRoot, templateId, stance string, frame int) error {
	bodyAnchor := Anchor{X: CanvasWidth / 2, Y: FootRow - 4}

	parts, err := listFrameParts(assetsRoot, templateId, stance, frame)
	if err != nil {
		return err
	}
	bodyMeta, hasBody := loadOrEmpty(assetsRoot, templateId, stance, frame, "body")
	if !hasBody {
		// Some sprites use "neck" or other names; fall back to first part.
		if len(parts) == 0 {
			return fmt.Errorf("%w: body sprite has no parts", ErrAssetsMissing)
		}
		bodyMeta, _ = loadOrEmpty(assetsRoot, templateId, stance, frame, parts[0])
	}

	type entry struct {
		part   string
		meta   PartMeta
		anchor Anchor
	}
	var entries []entry
	for _, part := range parts {
		meta, _ := loadOrEmpty(assetsRoot, templateId, stance, frame, part)
		anchor := Anchor{
			X: bodyAnchor.X - meta.Origin.X,
			Y: bodyAnchor.Y - meta.Origin.Y,
		}
		// All body parts share the body's origin frame — skip joint walk.
		_ = bodyMeta
		entries = append(entries, entry{part, meta, anchor})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return c.zIndex(entries[i].meta.Z) < c.zIndex(entries[j].meta.Z)
	})
	for _, e := range entries {
		if err := drawPart(canvas, assetsRoot, templateId, stance, frame, e.part, e.anchor); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compositor) zIndex(z string) int {
	for i, name := range c.zmap {
		if strings.EqualFold(name, z) {
			return i
		}
	}
	// Unknown z values sort to the back.
	return len(c.zmap)
}

func loadOrEmpty(assetsRoot, templateId, stance string, frame int, part string) (PartMeta, bool) {
	pm, err := LoadPartMeta(assetsRoot, templateId, stance, frame, part)
	if err != nil {
		return PartMeta{}, false
	}
	return pm, true
}

func listFrameParts(assetsRoot, templateId, stance string, frame int) ([]string, error) {
	dir := filepath.Join(assetsRoot, "character-parts", templateId, stance, strconv.Itoa(frame))
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrAssetsMissing, dir)
		}
		return nil, fmt.Errorf("readdir %s: %w", dir, err)
	}
	var out []string
	for _, e := range ents {
		name := e.Name()
		if strings.HasSuffix(name, ".png") {
			out = append(out, strings.TrimSuffix(name, ".png"))
		}
	}
	return out, nil
}

func drawPart(canvas *image.RGBA, assetsRoot, templateId, stance string, frame int, part string, anchor Anchor) error {
	path := filepath.Join(assetsRoot, "character-parts", templateId, stance, strconv.Itoa(frame), part+".png")
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open part %s: %w", path, err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("decode part %s: %w", path, err)
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	dr := image.Rect(anchor.X, anchor.Y, anchor.X+w, anchor.Y+h)
	draw.Draw(canvas, dr, img, image.Point{}, draw.Over)
	return nil
}
```

- [ ] **Step 2: Build a synthetic-fixture test for bare body**

`compositor_test.go`:

```go
package characterimage

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// writeSyntheticBody creates a 4x4 colored body sprite under the assets root.
func writeSyntheticBody(t *testing.T, root string) string {
	t.Helper()
	templateId := "00002000"
	frameDir := filepath.Join(root, "character-parts", templateId, "stand1", "0")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	pngFile, err := os.Create(filepath.Join(frameDir, "body.png"))
	if err != nil {
		t.Fatalf("create png: %v", err)
	}
	defer pngFile.Close()
	if err := png.Encode(pngFile, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	if err := os.WriteFile(filepath.Join(frameDir, "body.json"),
		[]byte(`{"origin":{"x":2,"y":3},"map":{"neck":{"x":0,"y":-3}},"z":"body","group":"skin"}`), 0o644); err != nil {
		t.Fatalf("write sidecar: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "character-parts", templateId, "info.json"),
		[]byte(`{"islot":"Bd","vslot":"Bd","cash":0}`), 0o644); err != nil {
		t.Fatalf("write info: %v", err)
	}
	return templateId
}

func writeSyntheticMaps(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "character-meta")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zmap.json"),
		[]byte(`["body","arm","head","hairOverHead"]`), 0o644); err != nil {
		t.Fatalf("zmap: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "smap.json"),
		[]byte(`{}`), 0o644); err != nil {
		t.Fatalf("smap: %v", err)
	}
}

func TestCompositeBareBody(t *testing.T) {
	root := t.TempDir()
	writeSyntheticMaps(t, root)
	writeSyntheticBody(t, root)

	c := NewCompositor()
	res, err := c.Composite(CompositeRequest{
		AssetsRoot: root,
		Skin:       0,
		Stance:     "stand1",
		Frame:      0,
		Resize:     1,
		Equipment:  map[int]int{},
	})
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	if res.Image.Bounds().Dx() != CanvasWidth || res.Image.Bounds().Dy() != CanvasHeight {
		t.Fatalf("dims = %v", res.Image.Bounds())
	}
	if res.EquippedSlotCount != 0 {
		t.Fatalf("expected 0 equipped, got %d", res.EquippedSlotCount)
	}
	// Body origin (2,3) must land at canvas (48, 120), so the body sprite
	// occupies x:[46..49], y:[117..120].
	checkColored(t, res.Image, 46, 117)
	checkColored(t, res.Image, 49, 120)
	// Outside the sprite, pixels are transparent.
	if a := res.Image.RGBAAt(0, 0).A; a != 0 {
		t.Fatalf("pixel (0,0) alpha = %d, want 0", a)
	}
}

func checkColored(t *testing.T, img *image.RGBA, x, y int) {
	t.Helper()
	c := img.RGBAAt(x, y)
	if c.A == 0 || c.R == 0 {
		t.Fatalf("pixel (%d,%d) = %+v — expected body color", x, y, c)
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor_test.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/testdata/.gitkeep
git commit -m "feat(characterimage): bare-body compositor with z-order"
```

### Task 5.9: Compositor — equipment blitting via joint tree

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor_test.go`

- [ ] **Step 1: Add equipment-blitting logic**

In `compositor.go`, add a new method and call it from `Composite` after `blitBody`:

```go
// jointForSlot maps a render slot to the joint name on the body via which the
// equipment attaches. Slots not in this map are skipped.
var jointForSlot = map[int]string{
	-1:  "neck",  // hat — anchored via neck through head sprite chain (simplified: treat as neck)
	-2:  "neck",  // face
	-3:  "neck",  // eye accessory
	-4:  "neck",  // earrings
	-5:  "navel", // top
	-6:  "navel", // bottom
	-7:  "navel", // shoes (uses navel as a stand-in; v83 sprites use foot — refine in fixture if needed)
	-8:  "navel", // gloves (refined to "hand" in detail; navel is fallback)
	-9:  "navel", // cape
	-10: "hand",  // shield
	-11: "hand",  // weapon
	-12: "navel", // ring (no visual today — kept for completeness)
}

// blitEquipment iterates equipment in zmap order, resolves each part's joint
// anchor against the body, and blits the part canvases into `canvas`.
func (c *Compositor) blitEquipment(canvas *image.RGBA, req CompositeRequest, equipment map[int]int, stance string) error {
	bodyAnchor := Anchor{X: CanvasWidth / 2, Y: FootRow - 4}
	bodyTemplate := bodyTemplateId(req.IsMale, mustSkin(req.Skin))

	type entry struct {
		templateId string
		part       string
		meta       PartMeta
		anchor     Anchor
		zIdx       int
	}
	var entries []entry

	add := func(templateId string, jointFromBody string) error {
		parts, err := listFrameParts(req.AssetsRoot, templateId, stance, req.Frame)
		if err != nil {
			return err
		}
		for _, part := range parts {
			meta, ok := loadOrEmpty(req.AssetsRoot, templateId, stance, req.Frame, part)
			if !ok {
				continue
			}
			bodyJointMeta, _ := loadOrEmpty(req.AssetsRoot, bodyTemplate, stance, req.Frame, "body")
			anchor := ResolveAnchor(bodyAnchor, bodyJointMeta, meta, jointFromBody)
			entries = append(entries, entry{
				templateId: templateId, part: part, meta: meta, anchor: anchor,
				zIdx: c.zIndex(meta.Z),
			})
		}
		return nil
	}

	// Hair / face anchored via neck.
	if req.Hair != 0 {
		if err := add(strconv.Itoa(req.Hair), "neck"); err != nil {
			return err
		}
	}
	if req.Face != 0 {
		if err := add(strconv.Itoa(req.Face), "neck"); err != nil {
			return err
		}
	}
	for slot, id := range equipment {
		joint, ok := jointForSlot[slot]
		if !ok {
			continue
		}
		if err := add(strconv.Itoa(id), joint); err != nil {
			return err
		}
	}

	sort.SliceStable(entries, func(i, j int) bool { return entries[i].zIdx < entries[j].zIdx })
	for _, e := range entries {
		if err := drawPart(canvas, req.AssetsRoot, e.templateId, stance, req.Frame, e.part, e.anchor); err != nil {
			return err
		}
	}
	return nil
}

// mustSkin returns the WZ id for a validated internal skin (caller ensures
// validity via MapInternalSkin upstream).
func mustSkin(internal int) int {
	id, _ := MapInternalSkin(internal)
	return id
}
```

In `Composite`, replace the line `// Equipment blitting comes in Task 5.9.` with:

```go
	if err := c.blitEquipment(canvas, req, filtered, stance); err != nil {
		return nil, err
	}
```

- [ ] **Step 2: Add a synthesised hat over body test**

Append to `compositor_test.go`:

```go
func writeSyntheticHat(t *testing.T, root string, hatId int) {
	t.Helper()
	tmpl := "00010000" // pretend hat templateId
	if hatId != 0 {
		// caller chose an explicit id
	}
	frameDir := filepath.Join(root, "character-parts", tmpl, "stand1", "0")
	if err := os.MkdirAll(frameDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 6, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 6; x++ {
			img.SetRGBA(x, y, color.RGBA{B: 200, G: 50, A: 255})
		}
	}
	f, _ := os.Create(filepath.Join(frameDir, "default.png"))
	defer f.Close()
	_ = png.Encode(f, img)
	_ = os.WriteFile(filepath.Join(frameDir, "default.json"),
		[]byte(`{"origin":{"x":3,"y":3},"map":{"neck":{"x":0,"y":0}},"z":"cap"}`), 0o644)
	_ = os.WriteFile(filepath.Join(root, "character-parts", tmpl, "info.json"),
		[]byte(`{"islot":"Cp","vslot":"Cp","cash":0}`), 0o644)
}

func TestCompositeWithHatBlitsAboveBody(t *testing.T) {
	root := t.TempDir()
	// zmap places "cap" above "body".
	if err := os.MkdirAll(filepath.Join(root, "character-meta"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(root, "character-meta", "zmap.json"),
		[]byte(`["body","arm","cap"]`), 0o644)
	_ = os.WriteFile(filepath.Join(root, "character-meta", "smap.json"), []byte(`{}`), 0o644)

	writeSyntheticBody(t, root)
	writeSyntheticHat(t, root, 10000)

	c := NewCompositor()
	res, err := c.Composite(CompositeRequest{
		AssetsRoot: root,
		Skin:       0,
		Stance:     "stand1",
		Frame:      0,
		Resize:     1,
		Equipment:  map[int]int{-1: 10000},
	})
	if err != nil {
		t.Fatalf("Composite: %v", err)
	}
	// The hat sprite should land near the body's neck on canvas. With body
	// origin (2,3) at (48,120), body.neck = (48,117). Hat origin (3,3) over
	// joint neck (0,0) means hat anchor = (45, 114). So pixel (45,114) is hat.
	c1 := res.Image.RGBAAt(45, 114)
	if c1.B == 0 {
		t.Fatalf("hat pixel missing at (45,114): %+v", c1)
	}
}
```

- [ ] **Step 3: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterimage/...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterimage/compositor_test.go
git commit -m "feat(characterimage): equipment blitting via joint tree"
```

---

## Phase 6 — Render handler + atomic write + route registration

### Task 6.1: JSON:API error body writer

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/error.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/error_test.go`

- [ ] **Step 1: Failing test**

`error_test.go`:

```go
package characterrender

import (
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestWriteErrorJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteError(rec, 400, ErrorBody{
		Code:   "unknown-template-id",
		Title:  "Equipment templateId not present",
		Detail: "templateId 1002357 not found",
		Meta:   map[string]any{"templateId": 1002357},
	})
	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/vnd.api+json" {
		t.Fatalf("content-type = %q", ct)
	}
	var got struct {
		Errors []struct {
			Status string         `json:"status"`
			Code   string         `json:"code"`
			Title  string         `json:"title"`
			Detail string         `json:"detail"`
			Meta   map[string]any `json:"meta"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Errors) != 1 {
		t.Fatalf("errors len = %d", len(got.Errors))
	}
	if got.Errors[0].Code != "unknown-template-id" {
		t.Fatalf("code = %q", got.Errors[0].Code)
	}
	if got.Errors[0].Status != strconv.Itoa(400) {
		t.Fatalf("status = %q", got.Errors[0].Status)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterrender/...`
Expected: FAIL — undefined `WriteError`, `ErrorBody`.

- [ ] **Step 3: Implementation**

`error.go`:

```go
package characterrender

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// ErrorBody is the JSON:API errors-array entry shape.
type ErrorBody struct {
	Code   string         `json:"code"`
	Title  string         `json:"title"`
	Detail string         `json:"detail,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type wireError struct {
	Status string         `json:"status"`
	Code   string         `json:"code"`
	Title  string         `json:"title"`
	Detail string         `json:"detail,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

// WriteError serialises one error and sets the response status. Multiple
// errors are unusual on the render path and intentionally not supported here.
func WriteError(w http.ResponseWriter, status int, body ErrorBody) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Errors []wireError `json:"errors"`
	}{
		Errors: []wireError{{
			Status: strconv.Itoa(status),
			Code:   body.Code,
			Title:  body.Title,
			Detail: body.Detail,
			Meta:   body.Meta,
		}},
	})
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./characterrender/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/error.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/error_test.go
git commit -m "feat(characterrender): JSON:API error body writer"
```

### Task 6.2: Path parser

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/path.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/path_test.go`

- [ ] **Step 1: Failing test**

`path_test.go`:

```go
package characterrender

import "testing"

func TestParseRenderPath(t *testing.T) {
	got, err := ParseRenderPath(map[string]string{
		"tenant":  "ec876921-aaaa-bbbb-cccc-deadbeef0000",
		"region":  "GMS",
		"version": "83.1",
		"hash":    "abcdef1234567890",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Tenant != "ec876921-aaaa-bbbb-cccc-deadbeef0000" {
		t.Fatalf("tenant: %s", got.Tenant)
	}
	if got.Region != "GMS" {
		t.Fatalf("region: %s", got.Region)
	}
	if got.MajorVersion != 83 || got.MinorVersion != 1 {
		t.Fatalf("version: %d.%d", got.MajorVersion, got.MinorVersion)
	}
	if got.Hash != "abcdef1234567890" {
		t.Fatalf("hash: %s", got.Hash)
	}
}

func TestParseRenderPathRejectsBadHash(t *testing.T) {
	_, err := ParseRenderPath(map[string]string{
		"tenant": "t", "region": "GMS", "version": "83.1", "hash": "ZZZZ",
	})
	if err == nil {
		t.Fatal("expected error on bad hash")
	}
}

func TestParseRenderPathRejectsBadVersion(t *testing.T) {
	_, err := ParseRenderPath(map[string]string{
		"tenant": "t", "region": "GMS", "version": "abc", "hash": "abcdef1234567890",
	})
	if err == nil {
		t.Fatal("expected error on bad version")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./characterrender/...`
Expected: FAIL.

- [ ] **Step 3: Implementation**

`path.go`:

```go
package characterrender

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hashPattern = regexp.MustCompile(`^[a-f0-9]{16}$`)

// RenderPath is the parsed path component of a render request.
type RenderPath struct {
	Tenant       string
	Region       string
	MajorVersion uint16
	MinorVersion uint16
	Hash         string
}

// ParseRenderPath validates the gorilla/mux path vars produced by the route
// `/render/{tenant}/{region}/{version}/{hash}.png`. Hash must be 16 lowercase
// hex chars; version must be `MAJOR.MINOR` integers.
func ParseRenderPath(vars map[string]string) (RenderPath, error) {
	tenant := vars["tenant"]
	region := vars["region"]
	version := vars["version"]
	hash := vars["hash"]
	if tenant == "" || region == "" || version == "" || hash == "" {
		return RenderPath{}, fmt.Errorf("missing path component")
	}
	if !hashPattern.MatchString(hash) {
		return RenderPath{}, fmt.Errorf("invalid hash %q", hash)
	}
	parts := strings.SplitN(version, ".", 2)
	if len(parts) != 2 {
		return RenderPath{}, fmt.Errorf("invalid version %q", version)
	}
	major, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return RenderPath{}, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return RenderPath{}, fmt.Errorf("invalid minor version: %w", err)
	}
	return RenderPath{
		Tenant:       tenant,
		Region:       region,
		MajorVersion: uint16(major),
		MinorVersion: uint16(minor),
		Hash:         hash,
	}, nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./characterrender/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/path.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/path_test.go
git commit -m "feat(characterrender): parse render path"
```

### Task 6.3: Query parser

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/query.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/query_test.go`

- [ ] **Step 1: Failing test**

`query_test.go`:

```go
package characterrender

import (
	"net/url"
	"reflect"
	"testing"
)

func TestParseRenderQueryDefaults(t *testing.T) {
	got, err := ParseRenderQuery(url.Values{
		"skin": {"0"}, "hair": {"30030"}, "face": {"20000"},
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Stance != "stand1" || got.Frame != 0 || got.Resize != 2 {
		t.Fatalf("defaults wrong: %+v", got)
	}
	if !reflect.DeepEqual(got.Items, []int{}) && got.Items != nil {
		t.Fatalf("items default should be empty: %+v", got.Items)
	}
}

func TestParseRenderQueryItems(t *testing.T) {
	got, err := ParseRenderQuery(url.Values{
		"skin": {"3"}, "hair": {"30030"}, "face": {"20000"},
		"stance": {"stand2"}, "frame": {"1"}, "resize": {"4"},
		"items": {"1442024,1002357,1402024"},
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Stance != "stand2" || got.Frame != 1 || got.Resize != 4 {
		t.Fatalf("scalars: %+v", got)
	}
	want := []int{1442024, 1002357, 1402024}
	if !reflect.DeepEqual(got.Items, want) {
		t.Fatalf("items = %v, want %v", got.Items, want)
	}
}

func TestParseRenderQueryRejectsResizeOutOfRange(t *testing.T) {
	_, err := ParseRenderQuery(url.Values{
		"skin": {"0"}, "hair": {"30030"}, "face": {"20000"}, "resize": {"7"},
	})
	if err == nil {
		t.Fatal("expected error on resize=7")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./characterrender/...`
Expected: FAIL.

- [ ] **Step 3: Implementation**

`query.go`:

```go
package characterrender

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// RenderQuery is the parsed query component of a render request.
type RenderQuery struct {
	Skin   int
	Hair   int
	Face   int
	Stance string
	Frame  int
	Resize int
	Items  []int
}

// ParseRenderQuery extracts and validates the documented query params. It
// applies defaults: stance=stand1, frame=0, resize=2.
func ParseRenderQuery(q url.Values) (RenderQuery, error) {
	skin, err := requiredInt(q, "skin")
	if err != nil {
		return RenderQuery{}, err
	}
	hair, err := requiredInt(q, "hair")
	if err != nil {
		return RenderQuery{}, err
	}
	face, err := requiredInt(q, "face")
	if err != nil {
		return RenderQuery{}, err
	}
	stance := q.Get("stance")
	if stance == "" {
		stance = "stand1"
	}
	frame := 0
	if v := q.Get("frame"); v != "" {
		f, err := strconv.Atoi(v)
		if err != nil || f < 0 {
			return RenderQuery{}, fmt.Errorf("invalid frame %q", v)
		}
		frame = f
	}
	resize := 2
	if v := q.Get("resize"); v != "" {
		r, err := strconv.Atoi(v)
		if err != nil || r < 1 || r > 4 {
			return RenderQuery{}, fmt.Errorf("invalid resize %q", v)
		}
		resize = r
	}
	items, err := parseItemsCSV(q.Get("items"))
	if err != nil {
		return RenderQuery{}, err
	}
	return RenderQuery{
		Skin: skin, Hair: hair, Face: face,
		Stance: stance, Frame: frame, Resize: resize, Items: items,
	}, nil
}

func requiredInt(q url.Values, name string) (int, error) {
	v := q.Get(name)
	if v == "" {
		return 0, fmt.Errorf("missing %s", name)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", name, v)
	}
	return n, nil
}

func parseItemsCSV(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}
	out := []int{}
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		n, err := strconv.Atoi(tok)
		if err != nil {
			return nil, fmt.Errorf("invalid items entry %q", tok)
		}
		out = append(out, n)
	}
	return out, nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./characterrender/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/query.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/query_test.go
git commit -m "feat(characterrender): parse render query"
```

### Task 6.4: Atomic write

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/write.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/write_test.go`

- [ ] **Step 1: Failing test**

`write_test.go`:

```go
package characterrender

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAtomicWritePNGProducesFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "abc.png")
	data := []byte("\x89PNG\r\n\x1a\nfake")

	if err := AtomicWritePNG(target, bytes.NewReader(data)); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("content mismatch")
	}
}

func TestAtomicWritePNGConcurrent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "abc.png")
	data := []byte("\x89PNG\r\n\x1a\nfake-concurrent")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := AtomicWritePNG(target, bytes.NewReader(data)); err != nil {
				t.Errorf("write: %v", err)
			}
		}()
	}
	wg.Wait()

	f, err := os.Open(target)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	got, _ := io.ReadAll(f)
	if !bytes.Equal(got, data) {
		t.Fatalf("content mismatch after concurrent writes")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./characterrender/...`
Expected: FAIL — undefined `AtomicWritePNG`.

- [ ] **Step 3: Implementation**

`write.go`:

```go
package characterrender

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AtomicWritePNG writes `r` to `dst` such that no reader ever observes
// partial bytes. Concurrent writes for the same `dst` produce identical
// results when their inputs are identical (last-rename wins).
func AtomicWritePNG(dst string, r io.Reader) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(dst)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := io.Copy(tmp, r); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("copy: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./characterrender/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/write.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/write_test.go
git commit -m "feat(characterrender): atomic PNG write"
```

### Task 6.5: Observability span + counters

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/otel.go`

- [ ] **Step 1: Write the file**

```go
package characterrender

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	renderTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "character_render_total",
		Help: "Total number of character renders served, labelled by stance and two-handed override.",
	}, []string{"stance", "two_handed_override"})

	renderErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "character_render_errors_total",
		Help: "Total number of character render errors, labelled by reason code.",
	}, []string{"reason"})

	renderDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "character_render_duration_ms",
		Help:    "Render duration in milliseconds (cache miss path only).",
		Buckets: []float64{50, 100, 200, 300, 500, 750, 1000, 1500, 2000, 3000},
	})
)

// IncrementRender records a successful render outcome.
func IncrementRender(stance string, override bool) {
	tag := "false"
	if override {
		tag = "true"
	}
	renderTotal.WithLabelValues(stance, tag).Inc()
}

// IncrementError records a failed render outcome.
func IncrementError(reason string) { renderErrors.WithLabelValues(reason).Inc() }

// ObserveDurationMs records a render duration sample.
func ObserveDurationMs(ms float64) { renderDuration.Observe(ms) }
```

- [ ] **Step 2: Verify the dep is available**

If `prometheus/client_golang` isn't already a transitive dep, add it:

```bash
cd services/atlas-wz-extractor/atlas.com/wz-extractor
go get github.com/prometheus/client_golang/prometheus@latest
```

Then build:

```bash
go build ./characterrender/...
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/otel.go services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod services/atlas-wz-extractor/atlas.com/wz-extractor/go.sum
git commit -m "feat(characterrender): observability counters"
```

### Task 6.6: Handler

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/doc.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/handler.go`
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/handler_test.go`

- [ ] **Step 1: doc.go**

```go
// Package characterrender wires HTTP handling to characterimage.
//
// The endpoint shape is documented in
// docs/tasks/task-043-character-render-service/api-contracts.md.
package characterrender
```

- [ ] **Step 2: handler.go**

```go
package characterrender

import (
	"atlas-wz-extractor/characterimage"
	"bytes"
	"errors"
	"image/png"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// Handler holds the dependencies a render handler needs.
type Handler struct {
	AssetsRoot string
	Compositor *characterimage.Compositor
}

// NewHandler returns a fully constructed Handler.
func NewHandler(assetsRoot string, c *characterimage.Compositor) *Handler {
	return &Handler{AssetsRoot: assetsRoot, Compositor: c}
}

// HandleRender is the http.HandlerFunc.
func (h *Handler) HandleRender(l logrus.FieldLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path, err := ParseRenderPath(mux.Vars(r))
		if err != nil {
			IncrementError("invalid-input")
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "invalid-input", Title: "Invalid path", Detail: err.Error(),
			})
			return
		}
		query, err := ParseRenderQuery(r.URL.Query())
		if err != nil {
			IncrementError("invalid-input")
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "invalid-input", Title: "Invalid query", Detail: err.Error(),
			})
			return
		}

		canonical := CanonicalLoadoutString(
			path.Tenant, path.Region, path.MajorVersion, path.MinorVersion,
			query.Skin, query.Hair, query.Face,
			query.Stance, query.Frame, query.Resize,
			query.Items,
		)
		expected := LoadoutHash(canonical)
		if expected != path.Hash {
			IncrementError("hash-mismatch")
			WriteError(w, http.StatusBadRequest, ErrorBody{
				Code: "hash-mismatch", Title: "URL hash does not match query",
				Meta: map[string]any{"expected": expected, "got": path.Hash},
			})
			return
		}

		assetsRoot := filepath.Join(h.AssetsRoot, path.Tenant, path.Region,
			strconv.FormatUint(uint64(path.MajorVersion), 10)+"."+strconv.FormatUint(uint64(path.MinorVersion), 10))

		req := characterimage.CompositeRequest{
			AssetsRoot: assetsRoot,
			Skin:       query.Skin,
			Hair:       query.Hair,
			Face:       query.Face,
			Equipment:  itemsToSlotMap(query.Items),
			Stance:     query.Stance,
			Frame:      query.Frame,
			Resize:     query.Resize,
			IsMale:     false, // gender selection deferred — see plan note
		}

		res, err := h.Compositor.Composite(req)
		if err != nil {
			h.writeCompositorError(w, l, err)
			return
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, res.Image); err != nil {
			IncrementError("compositor-error")
			WriteError(w, http.StatusInternalServerError, ErrorBody{
				Code: "compositor-error", Title: "PNG encode failed",
			})
			return
		}

		dst := filepath.Join(assetsRoot, "character", path.Hash+".png")
		if err := AtomicWritePNG(dst, bytes.NewReader(buf.Bytes())); err != nil {
			l.WithError(err).Errorf("atomic write %s", dst)
			IncrementError("compositor-error")
			WriteError(w, http.StatusInternalServerError, ErrorBody{
				Code: "compositor-error", Title: "Failed to persist render",
			})
			return
		}

		IncrementRender(res.ResolvedStance, res.TwoHandedOverride)
		ObserveDurationMs(float64(time.Since(start).Milliseconds()))

		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		w.Header().Set("ETag", "\""+path.Hash+"\"")
		w.Header().Set("X-Render-Cache", "miss")
		w.Header().Set("X-Render-Ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
		_, _ = w.Write(buf.Bytes())
	}
}

// itemsToSlotMap converts the sorted item list into a map keyed by a synthetic
// slot id derived from item-classification. The compositor's joint-tree only
// needs the slot grouping for joint resolution; the actual slot indices in
// the request URL are not preserved (they're hidden in the canonical hash).
//
// We assign slots:
//
//	1xxxxxxx item id -> slot derived from id/10000:
//	  100xxxx hat        -> -1
//	  101xxxx face acc   -> -2
//	  102xxxx eye acc    -> -3
//	  103xxxx earrings   -> -4
//	  104xxxx top        -> -5
//	  105xxxx overall    -> -5
//	  106xxxx bottom     -> -6
//	  107xxxx shoes      -> -7
//	  108xxxx gloves     -> -8
//	  109xxxx shield     -> -10
//	  110xxxx-114xxxx cape -> -9
//	  130xxxx-149xxxx weapon -> -11
//
// Items whose classifications fall outside these ranges are silently dropped.
func itemsToSlotMap(items []int) map[int]int {
	out := map[int]int{}
	for _, id := range items {
		slot, ok := slotForItem(id)
		if !ok {
			continue
		}
		out[slot] = id
	}
	return out
}

func slotForItem(id int) (int, bool) {
	c := id / 10000
	switch {
	case c == 100:
		return -1, true
	case c == 101:
		return -2, true
	case c == 102:
		return -3, true
	case c == 103:
		return -4, true
	case c == 104, c == 105:
		return -5, true
	case c == 106:
		return -6, true
	case c == 107:
		return -7, true
	case c == 108:
		return -8, true
	case c == 109:
		return -10, true
	case c >= 110 && c <= 114:
		return -9, true
	case c >= 130 && c <= 149:
		return -11, true
	}
	return 0, false
}

func (h *Handler) writeCompositorError(w http.ResponseWriter, l logrus.FieldLogger, err error) {
	switch {
	case errors.Is(err, characterimage.ErrUnknownTemplateId):
		IncrementError("unknown-template-id")
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "unknown-template-id", Title: "Equipment templateId not present in extract",
			Detail: err.Error(),
		})
	case errors.Is(err, characterimage.ErrInvalidStance):
		IncrementError("invalid-stance")
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "invalid-stance", Title: "Unknown stance",
			Meta: map[string]any{"supported": characterimage.SupportedStances()},
			Detail: err.Error(),
		})
	case errors.Is(err, characterimage.ErrFrameOutOfRange):
		IncrementError("frame-out-of-range")
		WriteError(w, http.StatusBadRequest, ErrorBody{
			Code: "frame-out-of-range", Title: "Frame index out of range",
			Detail: err.Error(),
		})
	case errors.Is(err, characterimage.ErrAssetsMissing):
		IncrementError("missing-asset")
		WriteError(w, http.StatusNotFound, ErrorBody{
			Code: "missing-asset", Title: "Required sprite missing from extract",
			Detail: err.Error(),
		})
	default:
		l.WithError(err).Error("compositor error")
		IncrementError("compositor-error")
		WriteError(w, http.StatusInternalServerError, ErrorBody{
			Code: "compositor-error", Title: "Compositor failed",
		})
	}
}
```

- [ ] **Step 3: handler_test.go — happy path with synthetic assets**

```go
package characterrender

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"atlas-wz-extractor/characterimage"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// makeAssetsRoot prepares a synthetic assets root with a body skin sprite.
func makeAssetsRoot(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	tenantPath := filepath.Join(root, "tenant-a", "GMS", "83.1")

	// character-meta
	if err := os.MkdirAll(filepath.Join(tenantPath, "character-meta"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	_ = os.WriteFile(filepath.Join(tenantPath, "character-meta", "zmap.json"),
		[]byte(`["body","arm"]`), 0o644)
	_ = os.WriteFile(filepath.Join(tenantPath, "character-meta", "smap.json"),
		[]byte(`{}`), 0o644)

	// body skin 0
	bodyDir := filepath.Join(tenantPath, "character-parts", "00002000", "stand1", "0")
	_ = os.MkdirAll(bodyDir, 0o755)
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	f, _ := os.Create(filepath.Join(bodyDir, "body.png"))
	defer f.Close()
	_ = png.Encode(f, img)
	_ = os.WriteFile(filepath.Join(bodyDir, "body.json"),
		[]byte(`{"origin":{"x":2,"y":3},"map":{"neck":{"x":0,"y":-3}},"z":"body"}`), 0o644)
	_ = os.WriteFile(filepath.Join(tenantPath, "character-parts", "00002000", "info.json"),
		[]byte(`{"islot":"Bd","vslot":"Bd","cash":0}`), 0o644)

	return root, tenantPath
}

func TestHandleRenderHappyPath(t *testing.T) {
	root, _ := makeAssetsRoot(t)

	c := characterimage.NewCompositor()
	h := NewHandler(root, c)

	canonical := CanonicalLoadoutString("tenant-a", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 1, nil)
	hash := LoadoutHash(canonical)

	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png",
		h.HandleRender(logrus.New())).Methods(http.MethodGet)

	url := "/api/wz/character/render/tenant-a/GMS/83.1/" + hash + ".png?skin=0&hair=30030&face=20000&stance=stand1&frame=0&resize=1&items="
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	r.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Fatalf("content-type = %q", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("cache-control = %q", cc)
	}
	if e := rec.Header().Get("ETag"); e != "\""+hash+"\"" {
		t.Fatalf("etag = %q", e)
	}

	// Verify the file landed on disk.
	cached := filepath.Join(root, "tenant-a", "GMS", "83.1", "character", hash+".png")
	if _, err := os.Stat(cached); err != nil {
		t.Fatalf("cached file missing: %v", err)
	}
}

func TestHandleRenderHashMismatch(t *testing.T) {
	root, _ := makeAssetsRoot(t)
	c := characterimage.NewCompositor()
	h := NewHandler(root, c)
	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png",
		h.HandleRender(logrus.New())).Methods(http.MethodGet)

	wrong := hex.EncodeToString(sha256.New().Sum(nil))[:16]
	url := "/api/wz/character/render/tenant-a/GMS/83.1/" + wrong + ".png?skin=0&hair=30030&face=20000"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))

	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"hash-mismatch"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestHandleRenderInvalidStance(t *testing.T) {
	root, _ := makeAssetsRoot(t)
	c := characterimage.NewCompositor()
	h := NewHandler(root, c)
	r := mux.NewRouter()
	r.HandleFunc("/api/wz/character/render/{tenant}/{region}/{version}/{hash}.png",
		h.HandleRender(logrus.New())).Methods(http.MethodGet)

	canonical := CanonicalLoadoutString("tenant-a", "GMS", 83, 1, 0, 30030, 20000, "warp", 0, 1, nil)
	hash := LoadoutHash(canonical)
	url := "/api/wz/character/render/tenant-a/GMS/83.1/" + hash + ".png?skin=0&hair=30030&face=20000&stance=warp&frame=0&resize=1&items="
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, url, nil))
	if rec.Code != 400 {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"invalid-stance"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	_ = strconv.IntSize // pacify import
}
```

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./characterrender/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/doc.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/handler.go services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/handler_test.go
git commit -m "feat(characterrender): HTTP handler"
```

### Task 6.7: Route registration

**Files:**
- Create: `services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/resource.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/main.go`

- [ ] **Step 1: resource.go**

```go
package characterrender

import (
	"atlas-wz-extractor/rest"
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

// InitResource registers the render route.
func InitResource(h *Handler) func(si jsonapi.ServerInformation) server.RouteInitializer {
	return func(si jsonapi.ServerInformation) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			register := rest.RegisterHandler(l)(si)
			ren := router.PathPrefix("/wz/character").Subrouter()
			ren.HandleFunc(
				"/render/{tenant}/{region}/{version}/{hash}.png",
				register("render_character", h.handleRenderBridge()),
			).Methods(http.MethodGet)
		}
	}
}

func (h *Handler) handleRenderBridge() rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return h.HandleRender(d.Logger())
	}
}
```

- [ ] **Step 2: Wire into main.go**

In `main.go`, after the `extraction.NewProcessor` line, add:

```go
	cren := characterrender.NewHandler(outputImgDir, characterimage.NewCompositor())
```

And modify the `server.New(l)` chain to add the new initializer **after** the existing one:

```go
	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath("/api/").
		SetPort(os.Getenv("REST_PORT")).
		SetReadTimeout(60 * time.Minute).
		SetWriteTimeout(60 * time.Minute).
		AddRouteInitializer(extraction.InitResource(p, tdm.WaitGroup(), extraction.Dirs{InputDir: inputDir, OutputXmlDir: outputXmlDir})(GetServer())).
		AddRouteInitializer(characterrender.InitResource(cren)(GetServer())).
		Run()
```

Add the imports at the top of `main.go`:

```go
	"atlas-wz-extractor/characterimage"
	"atlas-wz-extractor/characterrender"
```

- [ ] **Step 3: Build**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./...`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/resource.go services/atlas-wz-extractor/atlas.com/wz-extractor/main.go
git commit -m "feat(wz-extractor): register character render route"
```

---

## Phase 7 — nginx config + extraction-time wipe

### Task 7.1: nginx `try_files` block

**Files:**
- Modify: `services/atlas-assets/nginx.conf`

- [ ] **Step 1: Insert the new `location` blocks**

Replace the `location /` block in `services/atlas-assets/nginx.conf` so the file reads:

```nginx
worker_processes  1;

events {
  worker_connections  1024;
}

http {
  include       mime.types;
  default_type  application/octet-stream;

  sendfile        on;
  keepalive_timeout  65;

  server {
    listen       8080;
    server_name  _;

    root /usr/assets;

    # Character renders: try the cached PNG, fall back to atlas-wz-extractor on miss.
    location ~ ^/(?<ctenant>[^/]+)/(?<cregion>[^/]+)/(?<cver>[^/]+)/character/(?<chash>[a-f0-9]{16})\.png$ {
      try_files $uri @character_render;
      add_header Access-Control-Allow-Origin *;
      add_header Cache-Control "public, max-age=86400, immutable";
    }

    location @character_render {
      proxy_pass http://atlas-wz-extractor:8080/api/wz/character/render/$ctenant/$cregion/$cver/$chash.png$is_args$args;
      proxy_set_header Host $host;
      proxy_read_timeout 30s;
    }

    location / {
      try_files $uri =404;

      add_header Access-Control-Allow-Origin *;
      add_header Cache-Control "public, max-age=86400";
    }
  }
}
```

- [ ] **Step 2: Validate the syntax**

If a local nginx is available:

```bash
docker run --rm -v "$PWD/services/atlas-assets/nginx.conf:/etc/nginx/nginx.conf:ro" nginx:alpine nginx -t -c /etc/nginx/nginx.conf
```

Expected: `the configuration file /etc/nginx/nginx.conf syntax is ok`. If the test fails because `mime.types` isn't available in the included path, ignore that — the production image bundles it.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-assets/nginx.conf
git commit -m "feat(atlas-assets): proxy character/{hash}.png misses to atlas-wz-extractor"
```

### Task 7.2: Wipe rendered cache before extraction

**Files:**
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor.go`
- Modify: `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor_test.go`

- [ ] **Step 1: Failing test**

Append to `extraction/processor_test.go`:

```go
func TestRunExtractionWipesCharacterCache(t *testing.T) {
	tmp := t.TempDir()
	imgOut := filepath.Join(tmp, "out", "img", "tenant-a", "GMS", "83.1")
	cacheDir := filepath.Join(imgOut, "character")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "abcdef1234567890.png"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := wipeCharacterCache(imgOut); err != nil {
		t.Fatalf("wipe: %v", err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("expected cacheDir gone, stat err=%v", err)
	}
}
```

The test calls a package-private helper that doesn't exist yet — write the helper next.

- [ ] **Step 2: Add the wipe helper**

In `extraction/processor.go`, add at the bottom:

```go
// wipeCharacterCache removes the {imgOut}/character directory so a fresh
// extraction does not serve stale renders against newly extracted assets.
// Per the design, character-parts/ and character-meta/ are kept and
// overwritten in place by the extraction itself.
func wipeCharacterCache(imgOut string) error {
	target := filepath.Join(imgOut, "character")
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove %s: %w", target, err)
	}
	return nil
}
```

Add `"os"` to the import block if not already present.

- [ ] **Step 3: Call the helper at the start of `runExtraction`**

In `runExtraction`, immediately after the `wzFiles` glob (and before the file loop), add:

```go
	if !xmlOnly {
		if err := wipeCharacterCache(imgOutPath); err != nil {
			l.WithError(err).Warnf("Unable to wipe character cache.")
		}
	}
```

(`xmlOnly` already gates the image branch elsewhere — wiping only when image extraction is going to happen avoids surprise wipes during XML-only runs.)

- [ ] **Step 4: Run the tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./extraction/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor.go services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor_test.go
git commit -m "feat(wz-extractor): wipe character render cache on each extraction"
```

### Task 7.3: Whole-service build + test

**Files:** none

- [ ] **Step 1: Build the entire service**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go build ./...`
Expected: success.

- [ ] **Step 2: Run all service tests**

Run: `cd services/atlas-wz-extractor/atlas.com/wz-extractor && go test ./...`
Expected: PASS.

- [ ] **Step 3: Run atlas-constants tests**

Run: `cd libs/atlas-constants && go test ./...`
Expected: PASS.

- [ ] **Step 4: No commit needed unless build/test surfaced fixes**

If a fix was needed, commit with message describing the cause.

---

## Phase 8 — atlas-ui cutover

### Task 8.1: Add `js-sha256` dependency

**Files:**
- Modify: `services/atlas-ui/package.json`
- Modify: `services/atlas-ui/package-lock.json`

- [ ] **Step 1: Install**

```bash
cd services/atlas-ui
npm install js-sha256@^0.11.0
```

- [ ] **Step 2: Verify the dep resolves**

Run: `cd services/atlas-ui && npm ls js-sha256`
Expected: a single resolved version printed.

- [ ] **Step 3: Commit**

```bash
git add services/atlas-ui/package.json services/atlas-ui/package-lock.json
git commit -m "feat(atlas-ui): add js-sha256 for loadout hashing"
```

### Task 8.2: New `characterRender.service.ts`

**Files:**
- Create: `services/atlas-ui/src/services/api/characterRender.service.ts`
- Create: `services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts`
- Create: `services/atlas-ui/src/services/api/__tests__/loadout-hashes.json` (symlink or copied fixture)

- [ ] **Step 1: Copy the cross-language fixture**

The hash fixture is the source of truth — copy it into `__tests__` so vitest can read it:

```bash
cp services/atlas-wz-extractor/atlas.com/wz-extractor/characterrender/testdata/loadout-hashes.json \
   services/atlas-ui/src/services/api/__tests__/loadout-hashes.json
```

- [ ] **Step 2: Failing test**

`__tests__/characterRender.service.test.ts`:

```ts
import { describe, expect, it } from 'vitest';
import fixture from './loadout-hashes.json';
import {
  canonicalLoadoutString,
  loadoutHash,
  generateCharacterUrl,
  filterEquipment,
} from '../characterRender.service';

interface FixtureRow {
  tenant: string;
  region: string;
  majorVersion: number;
  minorVersion: number;
  skin: number;
  hair: number;
  face: number;
  stance: string;
  frame: number;
  resize: number;
  items: number[];
  canonical: string;
  expectedHash: string;
}

describe('characterRender canonical+hash parity', () => {
  for (const row of (fixture as { rows: FixtureRow[] }).rows) {
    it(`row ${row.tenant} ${row.stance} matches canonical`, () => {
      const canonical = canonicalLoadoutString(
        row.tenant, row.region, row.majorVersion, row.minorVersion,
        row.skin, row.hair, row.face,
        row.stance as any, row.frame, row.resize, row.items,
      );
      expect(canonical).toBe(row.canonical);
      expect(loadoutHash(canonical)).toBe(row.expectedHash);
    });
  }
});

describe('filterEquipment', () => {
  it('drops mount, pet, and cash slots', () => {
    const out = filterEquipment({
      '-1': 1002357,
      '-11': 1402024,
      '-14': 5000000,
      '-18': 1932000,
      '-19': 1932001,
      '-21': 1012000,
      '-101': 1002001,
      '-114': 1132001,
    });
    expect(out['-1']).toBe(1002357);
    expect(out['-11']).toBe(1402024);
    for (const slot of ['-14', '-18', '-19', '-21', '-101', '-114']) {
      expect(out[slot]).toBeUndefined();
    }
  });
});

describe('generateCharacterUrl', () => {
  it('builds the documented path/query shape', () => {
    const url = generateCharacterUrl(
      'tenant-a', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: { '-1': 1002357 } },
      { stance: 'stand1', frame: 0, resize: 2 },
    );
    expect(url.startsWith('/api/assets/tenant-a/GMS/83.1/character/')).toBe(true);
    expect(url).toMatch(/\/[a-f0-9]{16}\.png\?/);
    expect(url).toContain('skin=0');
    expect(url).toContain('hair=30030');
    expect(url).toContain('face=20000');
    expect(url).toContain('stance=stand1');
    expect(url).toContain('items=1002357');
  });

  it('sorts items so order does not change the URL', () => {
    const a = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: { '-1': 1442024, '-11': 1002357, '-5': 1402024 } },
      {});
    const b = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: { '-11': 1002357, '-5': 1402024, '-1': 1442024 } },
      {});
    expect(a).toBe(b);
  });
});
```

- [ ] **Step 3: Run to verify it fails**

Run: `cd services/atlas-ui && npm run test -- src/services/api/__tests__/characterRender.service.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 4: Implementation**

`src/services/api/characterRender.service.ts`:

```ts
import { sha256 } from 'js-sha256';
import type { Character } from '@/types/models/character';
import type { Asset } from '@/services/api/inventory.service';

export type Stance = 'stand1' | 'stand2' | 'walk1' | 'alert' | 'jump';

export interface CharacterLoadout {
  skin: number;
  hair: number;
  face: number;
  equipment: Record<string, number>;
}

export interface RenderOptions {
  stance?: Stance;
  frame?: number;
  resize?: number;
}

const CASH_SLOT_MIN = -114;
const CASH_SLOT_MAX = -101;
const FIXED_DROPPED_SLOTS = new Set([
  -14, // pet
  -18, -19, -20, // mount
  -21, -22, -23, -24, -25, -26, -27, -28, -29, -30, // pet rings
]);

export function filterEquipment(eq: Record<string, number>): Record<string, number> {
  const out: Record<string, number> = {};
  for (const [slot, id] of Object.entries(eq)) {
    const n = parseInt(slot, 10);
    if (FIXED_DROPPED_SLOTS.has(n)) continue;
    if (n >= CASH_SLOT_MIN && n <= CASH_SLOT_MAX) continue;
    out[slot] = id;
  }
  return out;
}

export function canonicalLoadoutString(
  tenant: string,
  region: string,
  major: number,
  minor: number,
  skin: number,
  hair: number,
  face: number,
  stance: Stance,
  frame: number,
  resize: number,
  items: number[],
): string {
  const sorted = [...items].sort((a, b) => a - b);
  return [
    tenant, region, `${major}.${minor}`,
    skin, hair, face,
    stance, frame, resize,
    sorted.join(','),
  ].join('|');
}

export function loadoutHash(canonical: string): string {
  return sha256(canonical).slice(0, 16);
}

export function generateCharacterUrl(
  tenant: string,
  region: string,
  major: number,
  minor: number,
  loadout: CharacterLoadout,
  options: RenderOptions = {},
): string {
  const opts: Required<RenderOptions> = {
    stance: options.stance ?? 'stand1',
    frame: options.frame ?? 0,
    resize: options.resize ?? 2,
  };
  const filtered = filterEquipment(loadout.equipment);
  const items = Object.values(filtered).sort((a, b) => a - b);
  const canonical = canonicalLoadoutString(
    tenant, region, major, minor,
    loadout.skin, loadout.hair, loadout.face,
    opts.stance, opts.frame, opts.resize, items,
  );
  const hash = loadoutHash(canonical);
  const params = new URLSearchParams({
    skin: String(loadout.skin),
    hair: String(loadout.hair),
    face: String(loadout.face),
    stance: opts.stance,
    frame: String(opts.frame),
    resize: String(opts.resize),
    items: items.join(','),
  });
  return `/api/assets/${tenant}/${region}/${major}.${minor}/character/${hash}.png?${params.toString()}`;
}

// characterToLoadout extracts the render-relevant fields from a Character +
// inventory pair. Slot keys are the negative-equipment-slot strings from the
// inventory asset model (e.g. "-1", "-11").
export function characterToLoadout(character: Character, inventory: Asset[]): CharacterLoadout {
  const equipment: Record<string, number> = {};
  for (const asset of inventory) {
    const slot = asset.attributes.slot;
    if (slot < 0) {
      equipment[String(slot)] = asset.attributes.templateId;
    }
  }
  return {
    skin: character.attributes.skinColor,
    hair: character.attributes.hair,
    face: character.attributes.face,
    equipment,
  };
}
```

- [ ] **Step 5: Run the tests**

Run: `cd services/atlas-ui && npm run test -- src/services/api/__tests__/characterRender.service.test.ts`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/services/api/characterRender.service.ts services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts services/atlas-ui/src/services/api/__tests__/loadout-hashes.json
git commit -m "feat(atlas-ui): characterRender.service URL builder + hash"
```

### Task 8.3: Update `useCharacterImage` and `CharacterRenderer`

**Files:**
- Modify: `services/atlas-ui/src/lib/hooks/useCharacterImage.ts`
- Modify: `services/atlas-ui/src/components/features/characters/CharacterRenderer.tsx`
- Modify: `services/atlas-ui/src/components/features/characters/OptimizedCharacterRenderer.tsx`

- [ ] **Step 1: Replace the hook's URL builder usage**

In `useCharacterImage.ts`, replace the import:

```ts
import { mapleStoryService } from '@/services/api/maplestory.service';
```

with:

```ts
import {
  generateCharacterUrl,
  characterToLoadout,
  type RenderOptions,
} from '@/services/api/characterRender.service';
```

Replace the `queryFn`:

```ts
    queryFn: async (): Promise<CharacterImageResult> => {
      const result = await mapleStoryService.generateCharacterImage(...);
      ...
      return result;
    },
```

with:

```ts
    queryFn: async (): Promise<CharacterImageResult> => {
      const url = generateCharacterUrl(
        character.tenant, character.region, character.majorVersion, character.minorVersion,
        { skin: character.skinColor, hair: character.hair, face: character.face, equipment: character.equipment },
        renderOptions as RenderOptions,
      );
      return {
        url,
        character,
        options: { ...renderOptions } as any,
        cached: false,
      };
    },
```

…and update the `MapleStoryCharacterData` type (in `src/types/models/maplestory.ts`) to require `tenant`, `region`, `majorVersion`, `minorVersion`. (If those fields don't exist yet, add them.)

Replace any other call to `mapleStoryService.generateCharacterImage` and `prefetchVariants` queryFns the same way.

- [ ] **Step 2: Update `CharacterRenderer.tsx`**

In `CharacterRenderer.tsx`, replace:

```ts
import { mapleStoryService } from '@/services/api/maplestory.service';
```

with:

```ts
import { characterToLoadout } from '@/services/api/characterRender.service';
```

And replace:

```ts
  const mapleStoryData = useMemo((): MapleStoryCharacterData =>
    mapleStoryService.characterToMapleStoryData(character, inventory),
    [character, inventory]
  );
```

with code that derives the loadout + per-tenant fields. The simplest path is to read tenant from `useTenant()`:

```ts
import { useTenant } from '@/context/tenant-context';
...
  const { activeTenant } = useTenant();
  const mapleStoryData = useMemo<MapleStoryCharacterData | null>(() => {
    if (!activeTenant) return null;
    const loadout = characterToLoadout(character, inventory);
    return {
      id: character.id,
      name: character.attributes.name,
      level: character.attributes.level,
      jobId: character.attributes.jobId,
      hair: loadout.hair,
      face: loadout.face,
      skinColor: loadout.skin,
      gender: character.attributes.gender,
      equipment: loadout.equipment,
      tenant: activeTenant.id,
      region: activeTenant.region,
      majorVersion: activeTenant.majorVersion,
      minorVersion: activeTenant.minorVersion,
    };
  }, [character, inventory, activeTenant]);
```

Bail out gracefully when `mapleStoryData` is null (skeleton).

- [ ] **Step 3: Update `OptimizedCharacterRenderer.tsx`**

Same import swap. Replace `mapleStoryService.characterToMapleStoryData` with `characterToLoadout` and inject the same tenant fields, or delete the wrapper entirely if it duplicates `CharacterRenderer.tsx`'s logic.

- [ ] **Step 4: Build + test**

Run: `cd services/atlas-ui && npm run lint && npm run test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/lib/hooks/useCharacterImage.ts services/atlas-ui/src/components/features/characters/CharacterRenderer.tsx services/atlas-ui/src/components/features/characters/OptimizedCharacterRenderer.tsx services/atlas-ui/src/types/models/maplestory.ts
git commit -m "feat(atlas-ui): consume new characterRender service"
```

### Task 8.4: Purge `maplestory.service.ts` and `maplestory.io` references

**Files:**
- Delete: `services/atlas-ui/src/services/api/maplestory.service.ts`
- Delete: `services/atlas-ui/src/services/api/__tests__/maplestory.service.test.ts`
- Modify: `services/atlas-ui/src/services/api/index.ts`
- Modify: `services/atlas-ui/src/components/features/characters/__tests__/CharacterRenderer.test.tsx`
- Modify: `services/atlas-ui/src/lib/utils/character-cache-sw.ts`
- Modify: `services/atlas-ui/public/sw-character-cache.js` (only if it references `maplestory.io`)

- [ ] **Step 1: Update tests to expect new URL shape**

In `CharacterRenderer.test.tsx`, replace every literal `https://maplestory.io/...` with `/api/assets/{tenant}/{region}/83.1/character/{hash}.png?...` shaped values. The simplest pattern: have the test mock `useCharacterImage` to return `{ imageUrl: 'mock-url' }` and assert `<img src='mock-url'>`. Concretely, search for the strings in the file and replace each occurrence:

- `'https://maplestory.io/api/character/test.png'` → `'/api/assets/tenant/GMS/83.1/character/abcdef1234567890.png'`
- `'https://maplestory.io/api/GMS/214/character/center/2002/30000:0,20000:0/stand1/0?resize=2'` → `'/api/assets/tenant/GMS/83.1/character/abcdef1234567890.png?skin=2&hair=30000&face=20000&stance=stand1&frame=0&resize=2&items='`

- [ ] **Step 2: Update `services/api/index.ts`**

Replace any `export * from './maplestory.service'` lines with:

```ts
export * from './characterRender.service';
```

If the index re-exported `mapleStoryService` by name elsewhere, remove those exports.

- [ ] **Step 3: Replace the hard-coded URL in `character-cache-sw.ts`**

Find and replace:

```ts
const url = `https://maplestory.io/api/GMS/214/character/center/${character.skinColor}/${character.hair}:0,${character.face}:0,${equipmentString}/${stance}/0?resize=${scale}`;
```

with code that calls the new builder:

```ts
import { generateCharacterUrl, characterToLoadout } from '@/services/api/characterRender.service';
...
const url = generateCharacterUrl(
  tenant, region, majorVersion, minorVersion,
  characterToLoadout(character as any, inventory),
  { stance, resize: scale },
);
```

(Pass `tenant`/`region`/`majorVersion`/`minorVersion` through whatever mechanism the cache-warmup code already uses; if it doesn't have them, accept them as new arguments and update callers.)

- [ ] **Step 4: Update `public/sw-character-cache.js` if needed**

```bash
grep -n "maplestory.io" services/atlas-ui/public/sw-character-cache.js
```

If lines exist, replace any URL-prefix allowlist to use `/api/assets/` instead.

- [ ] **Step 5: Delete `maplestory.service.ts` and its test**

```bash
git rm services/atlas-ui/src/services/api/maplestory.service.ts
git rm services/atlas-ui/src/services/api/__tests__/maplestory.service.test.ts
```

- [ ] **Step 6: Verify zero `maplestory.io` references**

Run:

```bash
grep -rn "maplestory.io" services/atlas-ui/src/ services/atlas-ui/public/
```

Expected: no output.

- [ ] **Step 7: Verify nothing imports the deleted module**

Run:

```bash
grep -rn "maplestory.service\|mapleStoryService" services/atlas-ui/src/
```

Expected: no output.

- [ ] **Step 8: Build + test**

Run: `cd services/atlas-ui && npm run lint && npm run build && npm run test`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add -A services/atlas-ui/
git commit -m "chore(atlas-ui): delete maplestory.service and remaining maplestory.io references"
```

### Task 8.5: Manual verification

**Files:** none

- [ ] **Step 1: Start the stack**

Whatever the dev workflow is (`docker compose up`, `make dev`, etc.). Open the UI at `http://localhost:3000`.

- [ ] **Step 2: Confirm no requests to `maplestory.io`**

Open DevTools → Network → filter `maplestory.io`. Navigate the accounts list, character detail, and ApplyPresetDialog. Expected: zero matching requests.

- [ ] **Step 3: Confirm cache hit on second load**

Reload an account detail page. The first load may show `X-Render-Cache: miss` headers; subsequent loads should be served by atlas-assets nginx with no `X-Render-Cache` header (because nginx serves the static file). Verify by inspecting Response Headers on the character image request.

- [ ] **Step 4: Confirm slot dropping**

Equip a mount on a test character and reload. The mount slot should not raise an error and the rendered character should still display. (Mount is silently dropped by `filterEquipment`.)

- [ ] **Step 5: Smoke check error states**

Hit the new endpoint manually with an invalid stance:

```bash
curl -i 'http://localhost:8080/api/assets/<tenantId>/GMS/83.1/character/0000000000000000.png?skin=0&hair=30030&face=20000&stance=warp'
```

Expected: 400 with JSON body containing `"hash-mismatch"` (because the stance changes the canonical string and thus the hash).

- [ ] **Step 6: No commit unless a fix surfaced**

If a manual issue produced a code change, commit with a `fix(...)` message describing the symptom and root cause.

---

## Acceptance criteria coverage map

| PRD criterion | Phase / Task |
|---|---|
| Character.wz dispatch emits worn-sprite assets + metadata | Phase 4 (4.4, 4.5) |
| Smap resolution table extracted and consumed | Phase 3 (3.2), Phase 5 (5.2, 5.8) |
| New render endpoint returns deterministic 96×128 / 192×256 PNGs | Phase 6 (6.6) |
| Cold render p95 < 500ms | Verified manually in Phase 8.5 |
| Cache hit served by atlas-assets nginx; wz-extractor not contacted | Phase 7 (7.1), Phase 8.5 |
| Character cache wiped at start of every extraction run | Phase 7 (7.2) |
| `mapleStoryService.generateCharacterUrl` replaced; no `maplestory.io` source refs | Phase 8 (8.2, 8.3, 8.4) |
| `frameMode='platform'` pixel-scan deleted | Confirmed not present (see context.md). No code change needed; acceptance still satisfied. |
| Browser Network tab shows zero `maplestory.io` requests | Phase 8.5 |
| Offline-egress cluster renders all loadouts | Phase 8.5 |
| Endpoint accepts the documented stance set; unknowns 400 | Phase 5 (5.4), Phase 6 (6.6) |
| Frame validation per stance | Phase 5 (5.4), Phase 6 (6.6) |
| Mount/pet/cash slots silently dropped | Phase 5 (5.5), Phase 8 (8.2) |
| Two-handed weapon overrides stance to `stand2` | Phase 5 (5.6) |
| OTel-style counters + duration histogram | Phase 6 (6.5, 6.6) |
| Tenant isolation via path namespacing | Path parsing (6.2) + handler assets-root (6.6) |
