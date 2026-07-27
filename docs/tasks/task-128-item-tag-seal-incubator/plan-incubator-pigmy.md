# Incubator Pigmy-Egg Correction — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the incubator behave as the real Pigmy-Egg mechanic — the sacrificed egg (`4170000–4170009`) selects a **per-region** reward pool, and the v95 client shows the correct region NPC — instead of one flat pool with a zeroed region id.

**Architecture:** Four small, sequenced changes plus a UI change: (1) the `INCUBATOR_RESULT` codec carries the egg id in v95's `gachaponItemID` slot; (2) the `incubator-rewards` tenant config gains an `eggId` dimension; (3) the channel reader filters the pool by egg; (4) the channel handler validates the sacrificed egg, rolls *that egg's* pool, and emits the egg id; (5) the admin UI groups pools by egg/region. A sixth phase (atlas-data ingest of `Etc.wz/incubatorInfo.img`) is **optional** — the client resolves the region NPC from its own copy, so it is not required for correct behavior.

**Tech Stack:** Go (libs/atlas-packet, atlas-channel, atlas-tenants), TypeScript/React (atlas-ui), JSON:API over api2go, tenant-context multi-tenancy.

## Global Constraints

- **Pigmy egg id range:** `4170000`–`4170009` (ETC "Pigmy Egg"; each id = one region). Eligible eggs = those with a configured pool; validate target templateId is in this range.
- **Incubator cash item:** `5060002` (`CashSlotItemTypeIncubator` = type 27).
- **`INCUBATOR_RESULT` body is version-gated (do NOT change per-version widths):** v83/84/87/jms = `int itemId, short count` (flat, IDA-verified); v95 = flat + `int gachaponItemID, int bonusItemID, int bonusCount`. Only `gachaponItemID` becomes non-zero (the egg id); the bonus pair stays `0`.
- **Client-interpreted wire values are config/data-resolved (DOM-25);** the egg id sent is the runtime sacrificed-item templateId, not a literal.
- **Reward pools are tenant-configured** (no WZ pigmy reward table exists); keyed per egg.
- Verify per CLAUDE.md: `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module; `docker buildx bake` only if a `go.mod` changed (none do here); atlas-ui `npm run build`+`test`, no new lint.

---

### Task 1: `IncubatorResult` carries the egg id (v95 `gachaponItemID`)

**Files:**
- Modify: `libs/atlas-packet/incubator/clientbound/result.go`
- Test: `libs/atlas-packet/incubator/clientbound/result_test.go`

**Interfaces:**
- Produces: `NewIncubatorResult(itemId uint32, count uint16, gachaponItemId uint32) IncubatorResult`; `IncubatorResult.GachaponItemId() uint32`. Encode writes `gachaponItemId` (not 0) into the v95 tail; flat body unchanged for v83/84/87/jms.

- [ ] **Step 1: Write the failing test** — add a v95 case asserting the egg id lands in the first tail int, and a v83 case asserting the flat body is unchanged.

```go
// libs/atlas-packet/incubator/clientbound/result_test.go
package clientbound_test

import (
	"context"
	"testing"

	inccb "github.com/Chronicle20/atlas/libs/atlas-packet/incubator/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func encode(t *testing.T, region string, major uint16, m inccb.IncubatorResult) []byte {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), region, major, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	return m.Encode(logrus.New(), ctx)(map[string]interface{}{})
}

func TestIncubatorResult_V95CarriesEggId(t *testing.T) {
	b := encode(t, "GMS", 95, inccb.NewIncubatorResult(2000000, 1, 4170005))
	// int itemId(4) + short count(2) + int gachaponItemID(4) + int bonusItemID(4) + int bonusCount(4) = 18
	if len(b) != 18 {
		t.Fatalf("v95 body len = %d, want 18", len(b))
	}
	// gachaponItemID at offset 6, little-endian 4170005 = 0x003F9CC5
	got := uint32(b[6]) | uint32(b[7])<<8 | uint32(b[8])<<16 | uint32(b[9])<<24
	if got != 4170005 {
		t.Fatalf("gachaponItemID = %d, want 4170005", got)
	}
}

func TestIncubatorResult_V83FlatUnchanged(t *testing.T) {
	b := encode(t, "GMS", 83, inccb.NewIncubatorResult(2000000, 1, 4170005))
	if len(b) != 6 {
		t.Fatalf("v83 body len = %d, want 6 (flat itemId+count)", len(b))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./incubator/clientbound/ -run IncubatorResult -v`
Expected: FAIL — `NewIncubatorResult` takes 2 args, not 3.

- [ ] **Step 3: Add the field + parameter and encode it**

```go
// result.go — struct
type IncubatorResult struct {
	itemId         uint32
	count          uint16
	gachaponItemId uint32
}

// NewIncubatorResult constructs an IncubatorResult. itemId <= 0 signals failure.
// gachaponItemId is the sacrificed Pigmy Egg id; the v95 client uses it to pick
// the region success NPC (GetGachaponSucessNpc). Pass 0 on failure/older versions.
func NewIncubatorResult(itemId uint32, count uint16, gachaponItemId uint32) IncubatorResult {
	return IncubatorResult{itemId: itemId, count: count, gachaponItemId: gachaponItemId}
}

func (m IncubatorResult) GachaponItemId() uint32 { return m.gachaponItemId }
```

```go
// result.go — Encode, replace the v95 tail block
		if t.Region() == "GMS" && t.MajorVersion() >= 95 {
			// v95 reads gachaponItemID (the sacrificed egg → region NPC) then a
			// bonus pair. Atlas rolls one reward, so the bonus pair stays zero.
			w.WriteInt(m.gachaponItemId)
			w.WriteInt(0)
			w.WriteInt(0)
		}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-packet && go test ./incubator/clientbound/ -run IncubatorResult -v`
Expected: PASS (both cases).

- [ ] **Step 5: Update the two existing callers** (they will fail to compile until updated; done fully in Task 4, but keep the lib green now by fixing the failure-path callers in the handler that pass `NewIncubatorResult(0, 0)` → `NewIncubatorResult(0, 0, 0)`).

Run: `grep -rn "NewIncubatorResult(" services/atlas-channel`
Expected: the `announceFailure` closure + the terminal success call in `character_cash_item_use.go`. Update both call sites to the 3-arg form (failure passes `0` as the egg id). Full handler wiring is Task 4.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/incubator/clientbound/result.go libs/atlas-packet/incubator/clientbound/result_test.go
git commit -m "feat(task-128): INCUBATOR_RESULT carries the egg id in the v95 gachaponItemID slot"
```

---

### Task 2: `incubator-rewards` tenant config gains `eggId`

**Files:**
- Modify: `services/atlas-tenants/atlas.com/tenants/configuration/rest.go` (`IncubatorRewardRestModel` + `ExtractIncubatorReward`)
- Modify: `services/atlas-tenants/configurations/incubator-rewards/*.json` (seed rows get an `eggId`)
- Test: `services/atlas-tenants/atlas.com/tenants/configuration/processor_test.go`

**Interfaces:**
- Produces: `IncubatorRewardRestModel` gains `EggId uint32 \`json:"eggId"\``; the extracted map carries `"eggId"`. Downstream JSON attribute name is `eggId`.

- [ ] **Step 1: Write the failing test** — extend the existing extract test to assert `eggId` round-trips.

```go
// processor_test.go — inside the existing IncubatorReward extract test
	restModel := configuration.IncubatorRewardRestModel{
		Id: "red-potion", ItemId: 2000000, Quantity: 50, Weight: 40, EggId: 4170000,
	}
	reward, err := configuration.ExtractIncubatorReward(restModel)
	if err != nil { t.Fatalf("ExtractIncubatorReward() unexpected error: %v", err) }
	if reward["eggId"] != uint32(4170000) {
		t.Errorf("reward[eggId] = %v, want 4170000", reward["eggId"])
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/ -run Incubator -v`
Expected: FAIL — `IncubatorRewardRestModel` has no `EggId`.

- [ ] **Step 3: Add the field + map it in Extract**

```go
// rest.go — add to IncubatorRewardRestModel
	EggId uint32 `json:"eggId"`
```

In `ExtractIncubatorReward`, add `"eggId": rm.EggId` to the returned map (match the existing key style used for itemId/quantity/weight).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-tenants/atlas.com/tenants && go test ./configuration/ -run Incubator -v`
Expected: PASS.

- [ ] **Step 5: Add `eggId` to the seed rows** — every file in `services/atlas-tenants/configurations/incubator-rewards/` gets `"eggId": 4170000` (put the existing default pool under the first region so a seed still yields a usable pool). Example:

```json
{
  "id": "red-potion",
  "type": "incubator-rewards",
  "attributes": { "eggId": 4170000, "itemId": 2000000, "quantity": 50, "weight": 40 }
}
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-tenants/atlas.com/tenants/configuration/rest.go services/atlas-tenants/atlas.com/tenants/configuration/processor_test.go services/atlas-tenants/configurations/incubator-rewards/
git commit -m "feat(task-128): incubator-rewards config gains per-egg (region) eggId"
```

---

### Task 3: Channel reader — per-egg pool

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/incubator/rest.go` (`RewardRestModel`, `Reward`, `Extract`)
- Modify: `services/atlas-channel/atlas.com/channel/incubator/processor.go` (`GetRewardsForEgg`)
- Test: `services/atlas-channel/atlas.com/channel/incubator/roll_test.go`

**Interfaces:**
- Consumes: Task 2's `eggId` JSON attribute.
- Produces: `Reward.EggId() uint32`; `Processor.GetRewardsForEgg(eggId uint32) ([]Reward, error)` — returns only rewards whose `EggId == eggId`. Existing `GetRewards()` is retained for callers/tests that want the full pool.

- [ ] **Step 1: Write the failing test** — `GetRewardsForEgg` filters by egg; `Reward.EggId()` round-trips.

```go
// roll_test.go
func TestExtract_CarriesEggId(t *testing.T) {
	r, err := incubator.Extract(incubator.RewardRestModel{Id: "x", ItemId: 2000000, Quantity: 50, Weight: 40, EggId: 4170005})
	if err != nil { t.Fatalf("Extract: %v", err) }
	if r.EggId() != 4170005 {
		t.Fatalf("EggId() = %d, want 4170005", r.EggId())
	}
}

func TestFilterByEgg(t *testing.T) {
	all := []incubator.Reward{
		mustExtract(t, 2000000, 4170000), mustExtract(t, 2000001, 4170005), mustExtract(t, 1302000, 4170005),
	}
	got := incubator.FilterByEgg(all, 4170005)
	if len(got) != 2 {
		t.Fatalf("FilterByEgg(4170005) len = %d, want 2", len(got))
	}
}

func mustExtract(t *testing.T, itemId, eggId uint32) incubator.Reward {
	t.Helper()
	r, err := incubator.Extract(incubator.RewardRestModel{Id: "r", ItemId: itemId, Quantity: 1, Weight: 1, EggId: eggId})
	if err != nil { t.Fatalf("Extract: %v", err) }
	return r
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./incubator/ -run 'EggId|FilterByEgg' -v`
Expected: FAIL — `EggId`/`FilterByEgg` undefined.

- [ ] **Step 3: Add the field, getter, Extract mapping, and filter**

```go
// rest.go — RewardRestModel gains:
	EggId uint32 `json:"eggId"`
// Reward gains:
	eggId uint32
func (r Reward) EggId() uint32 { return r.eggId }
// Extract sets eggId: rm.EggId (alongside the existing fields)
```

```go
// roll.go — pure filter helper
// FilterByEgg returns only the rewards configured for the given Pigmy Egg id.
func FilterByEgg(rewards []Reward, eggId uint32) []Reward {
	out := make([]Reward, 0, len(rewards))
	for _, r := range rewards {
		if r.EggId() == eggId {
			out = append(out, r)
		}
	}
	return out
}
```

```go
// processor.go — add to the interface and impl
// GetRewardsForEgg returns the reward pool for one Pigmy Egg (region).
GetRewardsForEgg(eggId uint32) ([]Reward, error)

func (p *ProcessorImpl) GetRewardsForEgg(eggId uint32) ([]Reward, error) {
	all, err := p.GetRewards()
	if err != nil {
		return nil, err
	}
	return FilterByEgg(all, eggId), nil
}
```

Add `GetRewardsForEgg` to `mock/processor.go` if a mock exists (mirror the existing `GetRewards` mock field).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./incubator/ -run 'EggId|FilterByEgg' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/incubator/
git commit -m "feat(task-128): channel incubator reader filters the pool per Pigmy Egg"
```

---

### Task 4: Handler — validate egg, roll region pool, emit egg id

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` (incubator arm, ~lines 252–300)
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go`

**Interfaces:**
- Consumes: `incubator.Processor.GetRewardsForEgg`, `incubator.PickWeighted`, `incubatorcb.NewIncubatorResult(itemId, count, gachaponItemId)`.
- Produces: on success, an `IncubatorUse` saga (unchanged steps) + `INCUBATOR_RESULT` carrying the sacrificed egg id.

- [ ] **Step 1: Write the failing test** — a unit test for a new pure helper `isPigmyEgg(templateId uint32) bool` (range guard), covering in/below/above/boundary.

```go
// character_cash_item_use_test.go
func TestIsPigmyEgg(t *testing.T) {
	cases := map[uint32]bool{
		4169999: false, 4170000: true, 4170005: true, 4170009: true, 4170010: false, 2000000: false,
	}
	for id, want := range cases {
		if got := isPigmyEgg(id); got != want {
			t.Errorf("isPigmyEgg(%d) = %v, want %v", id, got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run IsPigmyEgg -v`
Expected: FAIL — `isPigmyEgg` undefined.

- [ ] **Step 3: Add the guard + rewire the incubator arm**

Add near the cash-slot-type consts:

```go
const (
	pigmyEggMinId uint32 = 4170000
	pigmyEggMaxId uint32 = 4170009
)

// isPigmyEgg reports whether templateId is an incubatable Pigmy Egg (the client
// enforces this; the server re-checks so a crafted request can't sacrifice
// arbitrary items).
func isPigmyEgg(templateId uint32) bool {
	return templateId >= pigmyEggMinId && templateId <= pigmyEggMaxId
}
```

In the incubator arm, after `target` is resolved and before rolling, insert the egg guard, switch the pool lookup to the egg, and pass the egg id into every `IncubatorResult`:

```go
	eggId := target.TemplateId()
	if !isPigmyEgg(eggId) {
		l.Warnf("Character [%d] attempted to incubate non-egg item [%d].", s.CharacterId(), eggId)
		announceFailure()
		return
	}
	rewards, err := incubator.NewProcessor(l, ctx).GetRewardsForEgg(eggId)
	if err != nil || len(rewards) == 0 {
		l.Warnf("Character [%d] used incubator on egg [%d] with no reward pool.", s.CharacterId(), eggId)
		announceFailure()
		return
	}
```

Update `announceFailure` to pass the egg id (it is in scope after the guard; before the guard it is 0):

```go
	announceFailure := func(egg uint32) {
		_ = session.Announce(l)(ctx)(wp)(incubatorcb.IncubatorResultWriter)(incubatorcb.NewIncubatorResult(0, 0, egg).Encode)(s)
	}
```

Call `announceFailure(0)` for the pre-egg empty-slot path and `announceFailure(eggId)` after the egg is known. On the terminal success step, build the result with the egg id:

```go
	// terminal IncubatorResult step payload → NewIncubatorResult(reward.ItemId(), uint16(reward.Quantity()), eggId)
```

(Locate the existing `IncubatorResult` step builder in the saga and thread `eggId` through the payload used to construct `NewIncubatorResult`.)

- [ ] **Step 4: Run test + build to verify**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run IsPigmyEgg -v && go build ./...`
Expected: test PASS, build clean (all `NewIncubatorResult` call sites now 3-arg).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_test.go
git commit -m "feat(task-128): incubator handler validates the egg, rolls its region pool, emits the egg id"
```

---

### Task 5: atlas-ui — per-egg (region) reward pools

**Files:**
- Modify: `services/atlas-ui/src/services/api/incubator-rewards.service.ts` (add `eggId`)
- Modify: `services/atlas-ui/src/lib/schemas/incubator-rewards.schema.ts` (add `eggId`)
- Modify: `services/atlas-ui/src/pages/tenants-incubator-rewards-form.tsx` (group by egg, egg selector)
- Test: `services/atlas-ui/src/services/api/__tests__/incubator-rewards.service.test.ts`, `.../pages/__tests__/tenants-incubator-rewards-form.test.tsx`

**Interfaces:**
- Consumes: the `eggId` attribute from Task 2's resource.
- Produces: `IncubatorRewardAttributes` gains `eggId: number`; the form groups rows by region and requires an egg on create.

- [ ] **Step 1: Write the failing test** — the service test asserts a created reward includes `eggId` in the JSON:API envelope.

```ts
// incubator-rewards.service.test.ts — extend the create test
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (api.post as any).mockResolvedValue({ id: "r1", attributes: { eggId: 4170005, itemId: 2000000, quantity: 1, weight: 50 } });
    await incubatorRewardsService.create(t, { eggId: 4170005, itemId: 2000000, quantity: 1, weight: 50 });
    expect(api.post).toHaveBeenCalledWith(
      `/api/tenants/${t}/configurations/incubator-rewards`,
      { data: { type: "incubator-rewards", attributes: { eggId: 4170005, itemId: 2000000, quantity: 1, weight: 50 } } },
      undefined,
    );
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-ui && source ~/.nvm/nvm.sh && nvm use 22 && npx vitest run incubator-rewards.service`
Expected: FAIL — `IncubatorRewardAttributes` has no `eggId`.

- [ ] **Step 3: Add `eggId` to the type + schema, and an egg selector to the form**

```ts
// incubator-rewards.service.ts — interface IncubatorRewardAttributes
  eggId: number;
```

```ts
// incubator-rewards.schema.ts — add to the zod object
  eggId: z.coerce.number().int().min(4170000).max(4170009),
```

In `tenants-incubator-rewards-form.tsx`: add an egg/region `<Select>` (options `4170000`–`4170009` with region labels), include `eggId` in the create/edit payload, and group the table rows by `eggId` (region header per group). Region labels (interim): `4170000 Henesys, 4170001 Ellinia, 4170002 Perion, 4170003 Kerning City, 4170004 El Nath, 4170005 Ludibrium, 4170006 Orbis, 4170007 Aqua Road, 4170009 Nautilus`.

- [ ] **Step 4: Run test to verify it passes + build**

Run: `cd services/atlas-ui && npx vitest run incubator-rewards && npm run build`
Expected: PASS + build clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/services/api/incubator-rewards.service.ts services/atlas-ui/src/lib/schemas/incubator-rewards.schema.ts services/atlas-ui/src/pages/tenants-incubator-rewards-form.tsx services/atlas-ui/src/services/api/__tests__/incubator-rewards.service.test.ts services/atlas-ui/src/pages/__tests__/tenants-incubator-rewards-form.test.tsx
git commit -m "feat(task-128): atlas-ui incubator rewards are per-egg (region)"
```

---

### Task 6: Re-verify the `INCUBATOR_RESULT` v95 byte-fixture cell

**Files:**
- Modify: the v95 `INCUBATOR_RESULT` evidence/fixture under `docs/packets/audits/` (via the verify procedure)

- [ ] **Step 1:** Drive the single-cell verify procedure (`/verify-packet` → `packet-verifier`, `docs/packets/audits/VERIFYING_A_PACKET.md`) for `incubator/clientbound/IncubatorResult × gms_v95`, asserting the body now encodes `itemId, count, gachaponItemID(=egg), 0, 0` and matches the v95 client read order (`OnIncubatorResult @0xa00380`). Re-generate the matrix; commit the three artifacts together.

- [ ] **Step 2: Full verification** — `go test -race ./...` + `go vet ./...` clean in libs/atlas-packet, atlas-channel, atlas-tenants; atlas-ui `npm run build` + `npm run test` + no new lint; `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` clean; `superpowers:requesting-code-review`.

- [ ] **Step 3: Push** the branch (updates PR #909).

---

## Optional Phase 7 (deferred): ingest `Etc.wz/incubatorInfo.img`

Not required for correct behavior — the client resolves the region NPC from its own copy once the server sends the egg id (Task 1/4). Ingest is worthwhile only to (a) drive the *authoritative* egg→region labels in the UI, (b) enforce the eligible-egg set from data rather than the `4170000–4170009` guard, and (c) resolve `4170008` (real-egg-with-missing-string vs. not-a-live-egg). If pursued: add an atlas-data reader modelled on `services/atlas-data/atlas.com/data/cash/reader.go` that parses `incubatorInfo.img/{eggId}/{su,msg,usingAggScope}` and a `GET /data/incubator/{eggId}` resource; then have Task 4's guard and Task 5's labels source from it. Track as a follow-up task if not done here.

## Self-Review

- **Spec coverage:** design §1 (mechanic/eligibility) → Task 4 guard; §2 (`incubatorInfo.img`) → Phase 7 (deferred, justified); §3 (version-gated packet + egg-id bug) → Task 1 + Task 6; §4 (gap table) → Tasks 2–5; §5 (corrected design: ingest/config/handler/packet/UI) → Tasks 1–5 (+7); §6 (region map) → Task 5 labels; §7 (`4170008`) → Phase 7 + Task 4 range guard tolerates its absence.
- **Placeholder scan:** none — each code step shows the actual struct field, function body, or test.
- **Type consistency:** `NewIncubatorResult(itemId, count, gachaponItemId)` (Task 1) is used identically in Task 4; `EggId`/`eggId` naming consistent across Go (`EggId uint32`) and TS (`eggId: number`); `GetRewardsForEgg` defined in Task 3 and consumed in Task 4.
