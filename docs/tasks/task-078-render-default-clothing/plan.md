# Character Render — Default Clothing Fallback Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A character with no Top/Bottom/Overall equipped renders wearing gender-appropriate beginner clothing instead of a bare body, with the resolved gender folded into the loadout cache hash on both the Go service (`atlas-renders`) and the TS producer (`atlas-ui`).

**Architecture:** Inject a default beginner coat/pants id into the same `slot→id` equipment map an equipped item would occupy, so the default flows through the existing compositing/z-order/occlusion path with zero special-casing. Gender is resolved by a single pure idempotent function (`ResolveGender`) mirrored byte-for-byte in Go and TS: an explicit `0|1` wins, otherwise inferred from the face id via the v83 convention `(face/1000)%10 == 1 ⇒ female`. The resolved gender is appended as the final field of the canonical loadout string on both sides so UI-produced hashes pass the service's URL-hash validation.

**Tech Stack:** Go (atlas-renders, `libs/atlas-constants/item`), TypeScript/React (atlas-ui, Vitest, `js-sha256`, TanStack React Query).

---

## File Structure

Go — `services/atlas-renders/atlas.com/renders/character/`:
- `gender.go` — **new**: gender constants, default-clothing id constants, `ResolveGender`, `defaultCoat`, `defaultPants`. Import-free (pure).
- `gender_test.go` — **new**: `ResolveGender` / `defaultCoat` / `defaultPants` tables.
- `query.go` — **modify**: add `RenderQuery.Gender int`, parse optional `gender` query param.
- `hash.go` — **modify**: `CanonicalLoadoutString` gains a trailing `gender int` param.
- `handler.go` — **modify**: resolve gender once, feed resolved value into the canonical recompute used for URL-hash validation.
- `handler_test.go` — **modify**: update the 3 existing `CanonicalLoadoutString` call sites to the new signature; add gender parse + canonical-gender + determinism cases.
- `composite.go` — **modify**: add `topSlot`/`bottomSlot` constants + `applyDefaultClothing`, call it right after the equipment slot-map is built.
- `composite_test.go` — **new**: the four FR-2 injection cases + female ids.

TypeScript — `services/atlas-ui/src/`:
- `services/api/characterRender.service.ts` — **modify**: `CharacterLoadout.gender?`, new `resolveGender`, trailing-gender on `canonicalLoadoutString`, gender resolution + query-param emission in `generateCharacterUrl`, `gender` in `characterToLoadout`.
- `lib/hooks/useCharacterImage.ts` — **modify**: thread gender through all five builder sites; export `generateQueryKey` for testing.
- `services/api/__tests__/characterRender.service.test.ts` — **modify**: fixture interface + parity-loop call gain `gender`; add resolveGender table, canonical-trailing, generateCharacterUrl gender, and collision tests.
- `services/api/__tests__/loadout-hashes.json` — **regenerate**: every row gains `gender`; `canonical`/`expectedHash` recomputed; one female row added.
- `lib/hooks/__tests__/useCharacterImage.test.ts` — **new**: query-key hash == URL hash for a gendered character (guards builder drift).

No nginx/deploy/schema/`atlas-character` changes. The TS producer emits `/api/assets/.../character/<hash>.png?...&gender=<resolved>`; nginx forwards the query verbatim — `gender` is a query param, not a path component.

---

## Task 1: Go — gender resolution + default-clothing constants

**Files:**
- Create: `services/atlas-renders/atlas.com/renders/character/gender.go`
- Test: `services/atlas-renders/atlas.com/renders/character/gender_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-renders/atlas.com/renders/character/gender_test.go`:

```go
package character

import "testing"

func TestResolveGender(t *testing.T) {
	cases := []struct {
		name        string
		genderParam int
		face        int
		want        int
	}{
		{"explicit-male-wins-over-female-face", GenderMale, 21000, GenderMale},
		{"explicit-female-wins-over-male-face", GenderFemale, 20000, GenderFemale},
		{"infer-female-from-21xxx", GenderUnspecified, 21000, GenderFemale},
		{"infer-male-from-20xxx", GenderUnspecified, 20000, GenderMale},
		{"infer-male-from-zero-face", GenderUnspecified, 0, GenderMale},
		{"infer-male-from-negative-face", GenderUnspecified, -5, GenderMale},
		{"infer-male-from-30xxx", GenderUnspecified, 30030, GenderMale},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveGender(tc.genderParam, tc.face); got != tc.want {
				t.Fatalf("ResolveGender(%d, %d) = %d; want %d", tc.genderParam, tc.face, got, tc.want)
			}
		})
	}
}

func TestDefaultCoatPants(t *testing.T) {
	if defaultCoat(GenderMale) != DefaultCoatMale || defaultPants(GenderMale) != DefaultPantsMale {
		t.Fatalf("male defaults wrong: coat=%d pants=%d", defaultCoat(GenderMale), defaultPants(GenderMale))
	}
	if defaultCoat(GenderFemale) != DefaultCoatFemale || defaultPants(GenderFemale) != DefaultPantsFemale {
		t.Fatalf("female defaults wrong: coat=%d pants=%d", defaultCoat(GenderFemale), defaultPants(GenderFemale))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -run 'TestResolveGender|TestDefaultCoatPants' -v`
Expected: FAIL — compile error, `undefined: ResolveGender`, `GenderMale`, etc.

- [ ] **Step 3: Write minimal implementation**

Create `services/atlas-renders/atlas.com/renders/character/gender.go`:

```go
package character

// Gender selectors. GenderUnspecified is the RenderQuery sentinel meaning the
// optional `gender` query param was absent and gender must be inferred.
const (
	GenderMale        = 0
	GenderFemale      = 1
	GenderUnspecified = -1
)

// Default beginner clothing item ids injected into an empty clothing slot so a
// character is never rendered bare (PRD FR-1). These are the single source of
// truth; if asset verification (plan Task 8) shows an id is not ingested, swap
// it here (and only here) — the UI never names item ids.
const (
	DefaultCoatMale    = 1040036
	DefaultPantsMale   = 1060026
	DefaultCoatFemale  = 1041046
	DefaultPantsFemale = 1061039
)

// ResolveGender maps an optional gender selector plus a face id to a concrete
// 0 (male) / 1 (female) value. Precedence: an explicit 0/1 wins; otherwise the
// v83 face convention (faceId/1000)%10 == 1 ⇒ female; anything else ⇒ male. A
// non-positive / unknown face id resolves to male.
//
// Idempotent: ResolveGender(0, face) == 0 and ResolveGender(1, face) == 1 for
// any face, so the handler can resolve once for the hash and Composite can
// resolve again for injection and always agree.
func ResolveGender(genderParam, face int) int {
	if genderParam == GenderMale || genderParam == GenderFemale {
		return genderParam
	}
	if face > 0 && (face/1000)%10 == 1 {
		return GenderFemale
	}
	return GenderMale
}

func defaultCoat(gender int) int {
	if gender == GenderFemale {
		return DefaultCoatFemale
	}
	return DefaultCoatMale
}

func defaultPants(gender int) int {
	if gender == GenderFemale {
		return DefaultPantsFemale
	}
	return DefaultPantsMale
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -run 'TestResolveGender|TestDefaultCoatPants' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-renders/atlas.com/renders/character/gender.go services/atlas-renders/atlas.com/renders/character/gender_test.go
git commit -m "feat(atlas-renders): add gender resolution + default-clothing constants"
```

---

## Task 2: Go — parse the optional `gender` query param

**Files:**
- Modify: `services/atlas-renders/atlas.com/renders/character/query.go`
- Test: `services/atlas-renders/atlas.com/renders/character/handler_test.go`

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-renders/atlas.com/renders/character/handler_test.go`:

```go
func TestParseRenderQueryGender(t *testing.T) {
	base := func() url.Values {
		q := url.Values{}
		q.Set("skin", "0")
		q.Set("hair", "30000")
		q.Set("face", "20000")
		return q
	}

	t.Run("absent-is-unspecified", func(t *testing.T) {
		rq, err := ParseRenderQuery(base())
		if err != nil {
			t.Fatalf("ParseRenderQuery: %v", err)
		}
		if rq.Gender != GenderUnspecified {
			t.Errorf("absent gender = %d; want %d", rq.Gender, GenderUnspecified)
		}
	})

	for _, tc := range []struct {
		in   string
		want int
	}{{"0", GenderMale}, {"1", GenderFemale}} {
		t.Run("valid-"+tc.in, func(t *testing.T) {
			q := base()
			q.Set("gender", tc.in)
			rq, err := ParseRenderQuery(q)
			if err != nil {
				t.Fatalf("ParseRenderQuery: %v", err)
			}
			if rq.Gender != tc.want {
				t.Errorf("gender %q = %d; want %d", tc.in, rq.Gender, tc.want)
			}
		})
	}

	for _, bad := range []string{"2", "-1", "x", "1.0"} {
		t.Run("invalid-"+bad, func(t *testing.T) {
			q := base()
			q.Set("gender", bad)
			if _, err := ParseRenderQuery(q); err == nil {
				t.Fatalf("expected error for gender=%q", bad)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -run TestParseRenderQueryGender -v`
Expected: FAIL — compile error, `rq.Gender` undefined.

- [ ] **Step 3: Add the field and parsing**

In `services/atlas-renders/atlas.com/renders/character/query.go`, add the `Gender` field to the struct:

```go
type RenderQuery struct {
	Skin   int
	Hair   int
	Face   int
	Stance string
	Frame  int
	Resize int
	Items  []int
	Gender int // 0, 1, or GenderUnspecified(-1) when absent
}
```

In `ParseRenderQuery`, after the `items` parsing block and before the `return`, add:

```go
	gender := GenderUnspecified
	if v := q.Get("gender"); v != "" {
		g, err := strconv.Atoi(v)
		if err != nil || (g != GenderMale && g != GenderFemale) {
			return RenderQuery{}, fmt.Errorf("invalid gender %q", v)
		}
		gender = g
	}
```

And add `Gender: gender` to the returned struct literal:

```go
	return RenderQuery{
		Skin: skin, Hair: hair, Face: face,
		Stance: stance, Frame: frame, Resize: resize, Items: items,
		Gender: gender,
	}, nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -run TestParseRenderQueryGender -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-renders/atlas.com/renders/character/query.go services/atlas-renders/atlas.com/renders/character/handler_test.go
git commit -m "feat(atlas-renders): parse optional gender query param"
```

---

## Task 3: Go — append resolved gender to the canonical loadout hash

**Files:**
- Modify: `services/atlas-renders/atlas.com/renders/character/hash.go`
- Modify: `services/atlas-renders/atlas.com/renders/character/handler.go:81-84`
- Test: `services/atlas-renders/atlas.com/renders/character/handler_test.go`

Note: extending `CanonicalLoadoutString` ripples to every existing caller. This task updates the signature, the handler call, and the three existing test call sites together so the package always compiles.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-renders/atlas.com/renders/character/handler_test.go`:

```go
func TestCanonicalLoadoutStringGender(t *testing.T) {
	// Trailing gender field is present and last.
	got := CanonicalLoadoutString("T", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 2, nil, GenderFemale)
	want := "T|GMS|83.1|0|30030|20000|stand1|0|2||1"
	if got != want {
		t.Fatalf("canonical = %q; want %q", got, want)
	}
}

func TestCanonicalLoadoutGenderNoCollision(t *testing.T) {
	// Same empty loadout, different resolved gender → different hash.
	male := LoadoutHash(CanonicalLoadoutString("T", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 2, nil, GenderMale))
	female := LoadoutHash(CanonicalLoadoutString("T", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 2, nil, GenderFemale))
	if male == female {
		t.Fatalf("male and female empty-loadout hashes collide: %s", male)
	}
}

func TestHandlerCanonicalMirrorsUIGender(t *testing.T) {
	// Faithfully reproduces the handler's recomputation: a UI-style query that
	// carries the resolved gender must hash identically to what the UI produced
	// with that same resolved gender. (A full Handler() HTTP test would require
	// a MinIO storage + tenant-context harness this package does not have; the
	// handler's only gender-specific logic is ResolveGender + CanonicalLoadoutString,
	// exercised here directly.)
	q := url.Values{}
	q.Set("skin", "0")
	q.Set("hair", "30030")
	q.Set("face", "20000")
	q.Set("gender", "1")
	rq, err := ParseRenderQuery(q)
	if err != nil {
		t.Fatalf("ParseRenderQuery: %v", err)
	}
	g := ResolveGender(rq.Gender, rq.Face)
	service := LoadoutHash(CanonicalLoadoutString("T", "GMS", 83, 1, rq.Skin, rq.Hair, rq.Face, rq.Stance, rq.Frame, rq.Resize, rq.Items, g))
	ui := LoadoutHash(CanonicalLoadoutString("T", "GMS", 83, 1, 0, 30030, 20000, "stand1", 0, 2, nil, GenderFemale))
	if service != ui {
		t.Fatalf("service hash %s != UI hash %s", service, ui)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -run 'TestCanonicalLoadoutStringGender|TestCanonicalLoadoutGenderNoCollision|TestHandlerCanonicalMirrorsUIGender' -v`
Expected: FAIL — compile error, `CanonicalLoadoutString` called with too many arguments (signature still has 11 params).

- [ ] **Step 3: Extend the signature and update all callers**

In `services/atlas-renders/atlas.com/renders/character/hash.go`, change `CanonicalLoadoutString` to take a trailing `gender int` and append it:

```go
func CanonicalLoadoutString(
	tenant, region string,
	majorVersion, minorVersion uint16,
	skin, hair, face int,
	stance string,
	frame, resize int,
	items []int,
	gender int,
) string {
	sorted := append([]int(nil), items...)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.Itoa(id)
	}
	return fmt.Sprintf(
		"%s|%s|%d.%d|%d|%d|%d|%s|%d|%d|%s|%d",
		tenant, region, majorVersion, minorVersion,
		skin, hair, face, stance, frame, resize,
		strings.Join(parts, ","),
		gender,
	)
}
```

In `services/atlas-renders/atlas.com/renders/character/handler.go`, replace the canonical block (lines 81-84) with a resolve-then-canonical sequence:

```go
		g := ResolveGender(q.Gender, q.Face)
		canonical := CanonicalLoadoutString(
			urlTenant, urlRegion, t.MajorVersion(), t.MinorVersion(),
			q.Skin, q.Hair, q.Face, q.Stance, q.Frame, q.Resize, q.Items, g,
		)
```

In `services/atlas-renders/atlas.com/renders/character/handler_test.go`, update the three existing call sites in `TestLoadoutHashDeterministic` and `TestLoadoutHashLength` to pass a trailing `GenderMale`:

```go
				h1 := LoadoutHash(CanonicalLoadoutString("T", "GMS", 83, 1, 0, 30000, 20000, "stand1", 0, 2, tc.a, GenderMale))
				h2 := LoadoutHash(CanonicalLoadoutString("T", "GMS", 83, 1, 0, 30000, 20000, "stand1", 0, 2, tc.b, GenderMale))
```

```go
	h := LoadoutHash(CanonicalLoadoutString("T", "GMS", 83, 1, 0, 30000, 20000, "stand1", 0, 2, nil, GenderMale))
```

- [ ] **Step 4: Run the whole package to verify it passes**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -v`
Expected: PASS (all existing tests + the three new gender tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-renders/atlas.com/renders/character/hash.go services/atlas-renders/atlas.com/renders/character/handler.go services/atlas-renders/atlas.com/renders/character/handler_test.go
git commit -m "feat(atlas-renders): fold resolved gender into canonical loadout hash"
```

---

## Task 4: Go — inject default clothing into empty slots

**Files:**
- Modify: `services/atlas-renders/atlas.com/renders/character/composite.go`
- Test: `services/atlas-renders/atlas.com/renders/character/composite_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/atlas-renders/atlas.com/renders/character/composite_test.go`:

```go
package character

import "testing"

const (
	testTopID     = 1040002 // ClassificationTop
	testBottomID  = 1060002 // ClassificationBottom
	testOverallID = 1050002 // ClassificationOverall
)

func TestApplyDefaultClothing(t *testing.T) {
	t.Run("overall-suppresses-both", func(t *testing.T) {
		eq := map[int]int{topSlot: testOverallID}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != testOverallID {
			t.Errorf("overall overwritten: %d", eq[topSlot])
		}
		if _, ok := eq[bottomSlot]; ok {
			t.Errorf("overall must suppress default pants; got %d", eq[bottomSlot])
		}
	})

	t.Run("real-top-empty-bottom-injects-pants", func(t *testing.T) {
		eq := map[int]int{topSlot: testTopID}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != testTopID {
			t.Errorf("real top overwritten: %d", eq[topSlot])
		}
		if eq[bottomSlot] != DefaultPantsMale {
			t.Errorf("bottom = %d; want %d", eq[bottomSlot], DefaultPantsMale)
		}
	})

	t.Run("empty-top-real-bottom-injects-coat", func(t *testing.T) {
		eq := map[int]int{bottomSlot: testBottomID}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != DefaultCoatMale {
			t.Errorf("top = %d; want %d", eq[topSlot], DefaultCoatMale)
		}
		if eq[bottomSlot] != testBottomID {
			t.Errorf("real bottom overwritten: %d", eq[bottomSlot])
		}
	})

	t.Run("both-empty-injects-both-male", func(t *testing.T) {
		eq := map[int]int{}
		applyDefaultClothing(eq, GenderMale)
		if eq[topSlot] != DefaultCoatMale || eq[bottomSlot] != DefaultPantsMale {
			t.Errorf("male both = (%d,%d); want (%d,%d)", eq[topSlot], eq[bottomSlot], DefaultCoatMale, DefaultPantsMale)
		}
	})

	t.Run("both-empty-injects-both-female", func(t *testing.T) {
		eq := map[int]int{}
		applyDefaultClothing(eq, GenderFemale)
		if eq[topSlot] != DefaultCoatFemale || eq[bottomSlot] != DefaultPantsFemale {
			t.Errorf("female both = (%d,%d); want (%d,%d)", eq[topSlot], eq[bottomSlot], DefaultCoatFemale, DefaultPantsFemale)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -run TestApplyDefaultClothing -v`
Expected: FAIL — compile error, `topSlot`, `bottomSlot`, `applyDefaultClothing` undefined.

- [ ] **Step 3: Implement the helper and call it**

In `services/atlas-renders/atlas.com/renders/character/composite.go`, add slot constants and the helper. Place the constants next to the other `const` blocks (e.g. after the `hairPartClass`/`facePartClass`/`bodyPartClass` block) and the function near `FilterEquipment`:

```go
// topSlot / bottomSlot are the synthetic equipment-slot integers for the coat
// (Top/Overall) and pants (Bottom) halves, matching slotForItemID's numbering.
const (
	topSlot    = -5
	bottomSlot = -6
)

// applyDefaultClothing fills empty clothing slots with the gender's beginner
// coat/pants so a character is never rendered bare (PRD FR-2). An equipped
// Overall in the top slot covers both halves and suppresses both defaults. The
// two slots are otherwise independent. Defaults are injected as ordinary slot
// entries, so they flow through the existing compositing path unchanged.
func applyDefaultClothing(equipment map[int]int, gender int) {
	if id, ok := equipment[topSlot]; ok &&
		item.GetClassification(item.Id(uint32(id))) == item.ClassificationOverall {
		return
	}
	if _, ok := equipment[topSlot]; !ok {
		equipment[topSlot] = defaultCoat(gender)
	}
	if _, ok := equipment[bottomSlot]; !ok {
		equipment[bottomSlot] = defaultPants(gender)
	}
}
```

In `Composite`, immediately after the existing line `equipment := FilterEquipment(ItemsToSlotMap(q.Items))` (composite.go:264), add:

```go
	applyDefaultClothing(equipment, ResolveGender(q.Gender, q.Face))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./character/ -run TestApplyDefaultClothing -v`
Expected: PASS

- [ ] **Step 5: Run the full package + vet**

Run: `cd services/atlas-renders/atlas.com/renders && go test ./... && go vet ./...`
Expected: PASS, no vet findings.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-renders/atlas.com/renders/character/composite.go services/atlas-renders/atlas.com/renders/character/composite_test.go
git commit -m "feat(atlas-renders): inject default clothing into empty slots"
```

---

## Task 5: TS — gender plumbing in the render service, hook, fixture, and tests

This task changes `canonicalLoadoutString`'s signature (adds a required trailing `gender`), which ripples into `generateCharacterUrl` and the hook's `generateQueryKey`. To never leave the production build broken, **code lands in Commit A (build green), tests + fixture in Commit B (test green).**

**Files:**
- Modify: `services/atlas-ui/src/services/api/characterRender.service.ts`
- Modify: `services/atlas-ui/src/lib/hooks/useCharacterImage.ts`
- Regenerate: `services/atlas-ui/src/services/api/__tests__/loadout-hashes.json`
- Modify: `services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts`
- Create: `services/atlas-ui/src/lib/hooks/__tests__/useCharacterImage.test.ts`

### Commit A — production code (build stays green)

- [ ] **Step 1: Add `resolveGender` and thread gender through the service**

In `services/atlas-ui/src/services/api/characterRender.service.ts`:

Add `gender?` to the loadout interface:

```ts
export interface CharacterLoadout {
  skin: number;
  hair: number;
  face: number;
  equipment: Record<string, number>;
  gender?: number;
}
```

Add the resolver (mirrors Go `ResolveGender` byte-for-byte; `Math.floor` matches Go integer division for `face > 0`):

```ts
/**
 * Mirror of the Go service's ResolveGender. An explicit 0/1 wins; otherwise
 * infer from the face id via the v83 convention (face/1000)%10 === 1 ⇒ female.
 * A non-positive / unknown face resolves to male (0).
 */
export function resolveGender(gender: number | undefined, face: number): 0 | 1 {
  if (gender === 0 || gender === 1) return gender;
  if (face > 0 && Math.floor(face / 1000) % 10 === 1) return 1;
  return 0;
}
```

Add the trailing `gender` param to `canonicalLoadoutString`:

```ts
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
  gender: number,
): string {
  const sorted = [...items].sort((a, b) => a - b);
  return [
    tenant, region, `${major}.${minor}`,
    skin, hair, face,
    stance, frame, resize,
    sorted.join(','),
    gender,
  ].join('|');
}
```

Resolve gender in `generateCharacterUrl`, append it to the canonical string, and emit it as a query param. Replace the body from the `canonical` const through the `params` block:

```ts
  const filtered = filterEquipment(loadout.equipment);
  const items = Object.values(filtered).sort((a, b) => a - b);
  const gender = resolveGender(loadout.gender, loadout.face);
  const canonical = canonicalLoadoutString(
    tenant, region, major, minor,
    loadout.skin, loadout.hair, loadout.face,
    opts.stance, opts.frame, opts.resize, items, gender,
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
    gender: String(gender),
  });
```

Populate `gender` in `characterToLoadout`:

```ts
  return {
    skin: character.attributes.skinColor,
    hair: character.attributes.hair,
    face: character.attributes.face,
    equipment,
    gender: character.attributes.gender,
  };
```

- [ ] **Step 2: Thread gender through the hook's five builder sites**

In `services/atlas-ui/src/lib/hooks/useCharacterImage.ts`:

Add `resolveGender` to the import from the service:

```ts
import {
  generateCharacterUrl,
  filterEquipment,
  canonicalLoadoutString,
  loadoutHash,
  resolveGender,
  type RenderOptions,
  type Stance,
} from '@/services/api/characterRender.service';
```

In `generateQueryKey`, export it and resolve gender into the canonical call. Change the declaration to `export function generateQueryKey(...)` and update the `canonicalLoadoutString` call:

```ts
  const items = Object.values(filtered) as number[];
  const gender = resolveGender(character.gender, character.face);
  const canonical = canonicalLoadoutString(
    character.tenant,
    character.region,
    character.majorVersion,
    character.minorVersion,
    character.skinColor,
    character.hair,
    character.face,
    stance ?? 'stand1',
    frame,
    resize,
    items,
    gender,
  );
```

In each of the four `generateCharacterUrl(...)` call sites (the main `queryFn`, `prefetchVariants`, `preloadImages`, and `warmCache`), add `gender: character.gender,` to the loadout object literal. Each currently reads:

```ts
            {
              skin: character.skinColor,
              hair: character.hair,
              face: character.face,
              equipment: Object.fromEntries(
                Object.entries(character.equipment).map(([k, v]) => [k, v as number])
              ),
            },
```

Change each to:

```ts
            {
              skin: character.skinColor,
              hair: character.hair,
              face: character.face,
              equipment: Object.fromEntries(
                Object.entries(character.equipment).map(([k, v]) => [k, v as number])
              ),
              gender: character.gender,
            },
```

- [ ] **Step 3: Verify the production build is green**

Run: `cd services/atlas-ui && npm run build`
Expected: PASS (`tsc -b` + `vite build` succeed). Tests are excluded from the build; they are updated in Commit B.

- [ ] **Step 4: Commit A**

```bash
git add services/atlas-ui/src/services/api/characterRender.service.ts services/atlas-ui/src/lib/hooks/useCharacterImage.ts
git commit -m "feat(atlas-ui): thread resolved gender through render URL + cache key"
```

### Commit B — fixture + tests (tests stay green)

- [ ] **Step 5: Regenerate the cross-language hash fixture**

Create a throwaway generator `services/atlas-ui/scripts/gen-loadout-fixture.mjs` (deleted at the end of this step) that recomputes `canonical`/`expectedHash` with the trailing gender field using the same `js-sha256` the service uses:

```js
import { writeFileSync } from 'node:fs';
import { sha256 } from 'js-sha256';

// Inputs only; canonical + expectedHash are derived so they can never drift.
const inputs = [
  { tenant: 'tenant-a', region: 'GMS', majorVersion: 83, minorVersion: 1, skin: 0, hair: 30030, face: 20000, stance: 'stand1', frame: 0, resize: 2, items: [], gender: 0 },
  { tenant: 'tenant-a', region: 'GMS', majorVersion: 83, minorVersion: 1, skin: 0, hair: 30030, face: 20000, stance: 'stand1', frame: 0, resize: 2, items: [1002357], gender: 0 },
  { tenant: 'tenant-a', region: 'GMS', majorVersion: 83, minorVersion: 1, skin: 2, hair: 30030, face: 20000, stance: 'stand2', frame: 0, resize: 2, items: [1002357, 1402024, 1442024], gender: 0 },
  { tenant: 'tenant-b', region: 'GMS', majorVersion: 83, minorVersion: 1, skin: 0, hair: 30030, face: 20000, stance: 'walk1', frame: 1, resize: 4, items: [1002357], gender: 0 },
  { tenant: 'tenant-a', region: 'JMS', majorVersion: 83, minorVersion: 1, skin: 0, hair: 30030, face: 20000, stance: 'stand1', frame: 0, resize: 1, items: [], gender: 0 },
  { tenant: 'tenant-a', region: 'GMS', majorVersion: 83, minorVersion: 1, skin: 0, hair: 30030, face: 20000, stance: 'stand1', frame: 0, resize: 2, items: [], gender: 1 },
];

const rows = inputs.map((r) => {
  const sorted = [...r.items].sort((a, b) => a - b);
  const canonical = [
    r.tenant, r.region, `${r.majorVersion}.${r.minorVersion}`,
    r.skin, r.hair, r.face,
    r.stance, r.frame, r.resize,
    sorted.join(','),
    r.gender,
  ].join('|');
  return { ...r, canonical, expectedHash: sha256(canonical).slice(0, 16) };
});

writeFileSync(
  new URL('../src/services/api/__tests__/loadout-hashes.json', import.meta.url),
  JSON.stringify({ rows }, null, 2) + '\n',
);
console.log('wrote', rows.length, 'rows');
```

Run it, then delete it:

```bash
cd services/atlas-ui && node scripts/gen-loadout-fixture.mjs && rm scripts/gen-loadout-fixture.mjs && rmdir scripts 2>/dev/null || true
```

Expected: prints `wrote 6 rows`; `src/services/api/__tests__/loadout-hashes.json` now has a `gender` field on every row, a recomputed `canonical` ending in `|<gender>`, recomputed `expectedHash`, and one female (`gender: 1`) row.

- [ ] **Step 6: Update the service test (fixture interface + parity call) and add gender tests**

In `services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts`:

Add `gender` to the `FixtureRow` interface (after `items`):

```ts
  items: number[];
  gender: number;
  canonical: string;
  expectedHash: string;
```

Add `gender` to the import and pass `row.gender` in the parity loop's `canonicalLoadoutString` call:

```ts
import {
  canonicalLoadoutString,
  loadoutHash,
  generateCharacterUrl,
  filterEquipment,
  resolveGender,
  type Stance,
} from '../characterRender.service';
```

```ts
      const canonical = canonicalLoadoutString(
        row.tenant, row.region, row.majorVersion, row.minorVersion,
        row.skin, row.hair, row.face,
        row.stance as Stance, row.frame, row.resize, row.items, row.gender,
      );
```

Append these new describe blocks:

```ts
describe('resolveGender', () => {
  it('explicit 0/1 wins over face inference', () => {
    expect(resolveGender(0, 21000)).toBe(0);
    expect(resolveGender(1, 20000)).toBe(1);
  });
  it('infers female from 21xxx face', () => {
    expect(resolveGender(undefined, 21000)).toBe(1);
  });
  it('infers male from 20xxx face', () => {
    expect(resolveGender(undefined, 20000)).toBe(0);
  });
  it('non-positive / unknown face → male', () => {
    expect(resolveGender(undefined, 0)).toBe(0);
    expect(resolveGender(undefined, -5)).toBe(0);
    expect(resolveGender(undefined, 30030)).toBe(0);
  });
});

describe('canonicalLoadoutString gender', () => {
  it('appends resolved gender as the final field', () => {
    const c = canonicalLoadoutString('T', 'GMS', 83, 1, 0, 30030, 20000, 'stand1', 0, 2, [], 1);
    expect(c).toBe('T|GMS|83.1|0|30030|20000|stand1|0|2||1');
  });
});

describe('generateCharacterUrl gender', () => {
  it('emits a gender query param and a hash matching the canonical string', () => {
    const url = generateCharacterUrl(
      'tenant-a', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: {}, gender: 1 },
      { stance: 'stand1', frame: 0, resize: 2 },
    );
    expect(url).toContain('gender=1');
    const expected = loadoutHash(
      canonicalLoadoutString('tenant-a', 'GMS', 83, 1, 0, 30030, 20000, 'stand1', 0, 2, [], 1),
    );
    expect(url).toContain(`/${expected}.png?`);
  });

  it('male-face vs female-face empty loadout produce different hashes (no collision)', () => {
    const male = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 20000, equipment: {} }, {});
    const female = generateCharacterUrl('t', 'GMS', 83, 1,
      { skin: 0, hair: 30030, face: 21000, equipment: {} }, {});
    const hashOf = (u: string) => u.match(/\/([a-f0-9]{16})\.png\?/)?.[1];
    expect(hashOf(male)).toBeDefined();
    expect(hashOf(female)).toBeDefined();
    expect(hashOf(male)).not.toBe(hashOf(female));
  });
});
```

- [ ] **Step 7: Create the hook drift-guard test**

Create `services/atlas-ui/src/lib/hooks/__tests__/useCharacterImage.test.ts`. It guards that the React-Query key hash (`generateQueryKey`, builder #1) equals the URL hash (`generateCharacterUrl`, builder #2) for a gendered character — the highest-risk drift per design §10:

```ts
import { describe, expect, it } from 'vitest';
import { generateQueryKey } from '../useCharacterImage';
import { generateCharacterUrl } from '@/services/api/characterRender.service';
import type { MapleStoryCharacterData } from '@/types/models/maplestory';

function makeCharacter(overrides: Partial<MapleStoryCharacterData> = {}): MapleStoryCharacterData {
  return {
    id: 1,
    tenant: 'tenant-a',
    region: 'GMS',
    majorVersion: 83,
    minorVersion: 1,
    skinColor: 0,
    hair: 30030,
    face: 20000,
    gender: 0,
    equipment: {},
    ...overrides,
  } as MapleStoryCharacterData;
}

function urlHash(u: string): string | undefined {
  return u.match(/\/([a-f0-9]{16})\.png\?/)?.[1];
}

describe('useCharacterImage query-key vs URL hash parity', () => {
  for (const c of [makeCharacter(), makeCharacter({ face: 21000, gender: 1 })]) {
    it(`gender ${c.gender} face ${c.face} key hash equals URL hash`, () => {
      const keyHash = generateQueryKey(c)[1];
      const url = generateCharacterUrl(
        c.tenant, c.region, c.majorVersion, c.minorVersion,
        { skin: c.skinColor, hair: c.hair, face: c.face, equipment: c.equipment, gender: c.gender },
        {},
      );
      expect(keyHash).toBe(urlHash(url));
    });
  }
});
```

Note: if `MapleStoryCharacterData` has additional required fields the `as` cast does not satisfy at runtime usage, add the minimal fields the builders read (`tenant`, `region`, `majorVersion`, `minorVersion`, `skinColor`, `hair`, `face`, `gender`, `equipment`) — the cast covers the type, and only those fields are accessed by `generateQueryKey`/`generateCharacterUrl`.

- [ ] **Step 8: Run the TS test suite**

Run: `cd services/atlas-ui && npm run test`
Expected: PASS — parity fixture (6 rows incl. female), resolveGender table, canonical/url gender tests, and the hook drift-guard all green.

- [ ] **Step 9: Commit B**

```bash
git add services/atlas-ui/src/services/api/__tests__/loadout-hashes.json services/atlas-ui/src/services/api/__tests__/characterRender.service.test.ts services/atlas-ui/src/lib/hooks/__tests__/useCharacterImage.test.ts
git commit -m "test(atlas-ui): cover gender in render hash fixture, service, and hook"
```

---

## Task 6: Go verification gate

**Files:** none (verification only).

- [ ] **Step 1: Race + vet in atlas-renders**

Run: `cd services/atlas-renders/atlas.com/renders && go test -race ./... && go vet ./...`
Expected: PASS, no findings.

- [ ] **Step 2: Build the service**

Run: `cd services/atlas-renders/atlas.com/renders && go build ./...`
Expected: PASS.

- [ ] **Step 3: Redis key guard (repo root)**

Run: `cd "$(git rev-parse --show-toplevel)" && tools/redis-key-guard.sh`
Expected: clean (no new raw keyed go-redis usage; this change adds none).

- [ ] **Step 4: Docker bake**

Run: `cd "$(git rev-parse --show-toplevel)" && docker buildx bake atlas-renders`
Expected: PASS. (Per PRD §10 acceptance criteria; `atlas-renders` `go.mod` is untouched, but the bake is run to satisfy the acceptance gate.)

- [ ] **Step 5: Commit (only if any incidental fixes were needed)**

If steps 1-4 required no changes, skip. Otherwise:

```bash
git add -A && git commit -m "chore(atlas-renders): verification fixups"
```

---

## Task 7: TS verification gate

**Files:** none (verification only).

- [ ] **Step 1: Lint**

Run: `cd services/atlas-ui && npm run lint`
Expected: PASS (no new lint errors in `characterRender.service.ts`, `useCharacterImage.ts`, or the new/updated tests).

- [ ] **Step 2: Test**

Run: `cd services/atlas-ui && npm run test`
Expected: PASS.

- [ ] **Step 3: Build**

Run: `cd services/atlas-ui && npm run build`
Expected: PASS.

- [ ] **Step 4: Commit (only if any incidental fixes were needed)**

If steps 1-3 required no changes, skip. Otherwise:

```bash
git add -A && git commit -m "chore(atlas-ui): verification fixups"
```

---

## Task 8: Asset verification (release-blocking gate — PRD §9 / design §8)

**Files:** `services/atlas-renders/atlas.com/renders/character/gender.go` (only if an id must be swapped).

The feature is only *visibly* effective if the four default atlases exist in the renders bucket for the target region/version:

| Gender | Coat partClass/id | Pants partClass/id |
|--------|-------------------|--------------------|
| Male   | `Coat/1040036`    | `Pants/1060026`    |
| Female | `Coat/1041046`    | `Pants/1061039`    |

Atlas object keys (per `storage/atlas.go`): `<scope>/regions/<region>/versions/<version>/atlases/Coat/1040036.png` (+`.json`), and likewise for the other three.

- [ ] **Step 1: Probe a live/dev atlas-renders with empty-loadout renders**

Request an empty-loadout render for a male face and a female face against a live/dev atlas-renders (an empty `items=`, `gender=0` then `gender=1`). Confirm (a) the produced PNG shows clothing and (b) the service logs contain **no** `missing atlas: partClass=Coat` or `partClass=Pants` warning for the four ids.

Inspection options (from project memory `reference_atlas_data_wz_inspection` / `reference_observability`): live `GET` against atlas-renders via a throwaway curl pod with `TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION` headers, MinIO `atlas-renders` bucket listing for the four object keys, or `mcp__kubernetes__pods_log` on the atlas-renders pod to scan for the warn line.

Expected: clothed PNGs, no `missing atlas` warning for the four ids.

- [ ] **Step 2: If any id is absent, resolve and document**

If a warning fires for any id, that atlas is not ingested. Resolve by either:
- (a) trigger/await ingestion of the missing Coat/Pants id for that region/version, or
- (b) substitute an alternate beginner id confirmed present in the bucket. Because the ids are the single source of truth in `gender.go`, this is a one-line change to the relevant `Default*` constant (the UI never names item ids, so no TS change). If swapped, update the PRD §FR-1 table and re-run Tasks 4 and 6.

Record the outcome (verified-present, or swapped-to `<new id>`) in the branch's audit/PR notes so the §10 acceptance criterion "asset-verification resolved with evidence" is satisfied.

---

## Self-Review

**Spec coverage (design §11 / PRD §4 + §10):**
- FR-1 default ids as constants → Task 1 (`gender.go` `Default*` constants). ✅
- FR-2 per-slot independent fallback + overall suppression → Task 4 (`applyDefaultClothing`, four cases tested). ✅
- FR-3 gender resolution (param → face inference, hair not used) → Task 1 (`ResolveGender`). ✅
- FR-4 optional `gender` query param + 400 on invalid → Task 2. ✅
- FR-5 loadout hash includes gender (Go + TS) → Task 3 (Go) + Task 5 (TS canonical + query param). ✅
- FR-6 always clothe (no opt-out) → no opt-out code path introduced. ✅
- FR-7 missing-atlas degradation → unchanged warn-and-skip path (default injected as ordinary entry); verified in Task 8. ✅
- Determinism Go↔TS → mirrored `ResolveGender`/canonical + `TestHandlerCanonicalMirrorsUIGender` + parity fixture + hook drift-guard. ✅
- No-collision acceptance → `TestCanonicalLoadoutGenderNoCollision` (Go) + generateCharacterUrl collision test (TS). ✅
- Build/verify gates → Tasks 6 (Go: race/vet/build/redis-guard/bake) + 7 (TS: lint/test/build). ✅
- Asset-verification gate → Task 8. ✅

**Placeholder scan:** No TBD/TODO/"handle edge cases" — every code step shows full code; the fixture is regenerated by a complete script, not hand-filled hashes. ✅

**Type consistency:** `ResolveGender(genderParam, face)` / `resolveGender(gender, face)`, `topSlot`/`bottomSlot`, `applyDefaultClothing(equipment, gender)`, `defaultCoat`/`defaultPants`, `Default{Coat,Pants}{Male,Female}`, `CanonicalLoadoutString(..., items, gender)` / `canonicalLoadoutString(..., items, gender)`, `RenderQuery.Gender`, exported `generateQueryKey` — names are identical across every task that references them. ✅
