# Water of Life (Pet Revive) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop pets from being reaped when their lifespan expires, and let a player revive a dried-up pet by double-clicking a Water of Life (item `5180000`, classification 518), consuming exactly one item and resetting the pet's lifespan from WZ data.

**Architecture:** The client sends a bodyless `WATER_OF_LIFE` opcode. `atlas-channel` resolves both operands itself (a held classification-518 item; the most-recently-expired pet), then drives a two-step saga: `destroy_water_of_life` → `revive_pet`. `atlas-pets` derives the new expiration from the consumed item's WZ `info/life` and, inside one database transaction + outbox, writes its own row *and* buffers a `RESET_PET_EXPIRATION` cascade to `atlas-inventory`, which re-derives the same ceiling independently and rejects anything beyond it. The resulting asset `UPDATED` event rides the existing channel consumer back to the client as a full slot re-announce, which is what changes the tooltip. Separately, `atlas-asset-expiration` stops emitting expire commands for classification-500 assets so dolls exist at all.

**Tech Stack:** Go 1.x microservices (immutable models + Builder, `NewProcessor(l, ctx)`, `message.Buffer` + `message.Emit`, JSON:API via api2go, Kafka via `libs/atlas-kafka`, GORM + `database.ExecuteTransaction` + outbox), `libs/atlas-packet` codecs, `libs/atlas-saga` shared saga contract, tenant socket-config seed templates.

**Spec:** [`design.md`](design.md) (PRD: [`prd.md`](prd.md))

## Global Constraints

Copied verbatim from the spec and CLAUDE.md; every task's requirements implicitly include these.

- **Classification, never id allowlists.** The pet exemption is `item.GetClassification(id) != item.ClassificationPet` (500). The Water of Life is located by `item.ClassificationWaterOfLife` (518). No literal `518` or `500` survives in new code.
- **`info/life` is in DAYS.** Same WZ node `atlas-data`'s pet reader already parses. `5180000` in the GMS 83.1 extract is `life=90`. Absent or `0` ⇒ reject, consume nothing.
- **The channel sends no expiration anywhere.** It sends `{characterId, petId, sourceTemplateId}`. `atlas-pets` derives the absolute expiration; `atlas-inventory` re-derives the ceiling itself and **rejects, never clamps**.
- **No `EnableActions` on any path of the new handler.** `CWvsContext::SendWaterOfLife` (gms_v83 `0xa1dce6`) never arms the excl latch — `SetExclRequestSent` (`0xa0ebbc`) has exactly one caller, `SendConsumeCashItemUseRequest` (`0xa0ea6f`). Sending one would be a lie in the code.
- **The packet body is empty on all five versions.** Verified per-IDB: v83 `0xa1dce6`, v84 `0xa68f85`, v87 `0xab501c`, v92 `0x9c6f90`, v95 `0x9f28e0` — each is `COutPacket(op)` + `SendPacket` with zero `Encode*` calls. One codec, no version gates.
- **Opcodes are resolved from tenant configuration, never hard-coded.** Per `docs/packets/registry/gms_v{83,84,87,92,95}.yaml`: v83 `0x75`, v84 `0x75`, v87 `0x78`, v92 `0x80`, v95 `0x81`.
- **Applicable versions are gms_v83, v84, v87, v92, v95 only.** The handler is registered in exactly those five seed templates and in none of `template_gms_{12,48,61,72,79}_1.json` / `template_jms_185_1.json`. The `n-a` matrix cells (`gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`, `jms_v185`) stay `n-a`.
- **Template entries go at their sorted `opCode` position**, never appended next to a semantically-related entry (`tools/template-opcode-order-guard.sh`). Every handler entry carries a non-empty `validator` — a missing validator is silently dropped at dispatch-map build time.
- **No `// TODO`, stubs, or 501s in landed commits.**
- **Never write literal home/absolute paths into committed files.** Repo-relative paths only.
- **Test setup uses the project's Builder pattern.** No `*_testhelpers.go` files with test-only constructors.
- **No `go.mod` changes are required by this plan.** Every module that needs `libs/atlas-constants` already depends on it. If a task appears to need a new module dependency, stop and re-read — it almost certainly does not.

---

## File Structure

New files:

| Path | Responsibility |
|---|---|
| `libs/atlas-packet/pet/serverbound/water_of_life.go` | Empty-body `WATER_OF_LIFE` serverbound codec |
| `libs/atlas-packet/pet/serverbound/water_of_life_test.go` | Round-trip byte fixture (carries the `packet-audit:verify` markers) |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/expiration/checker_test.go` | `IsReapable` table |
| `services/atlas-asset-expiration/atlas.com/asset-expiration/character/processor_reap_test.go` | All three sweeps: expired non-pet emits, expired pet does not |
| `services/atlas-pets/atlas.com/pets/data/cash/{model,builder,rest,requests,processor}.go` | atlas-pets' cash-data client (`info/life`) |
| `services/atlas-pets/atlas.com/pets/data/cash/mock/processor.go` | Mock for the above |
| `services/atlas-pets/atlas.com/pets/pet/processor_revive_test.go` | Revive semantics + the three idempotency-gate rows |
| `services/atlas-pets/atlas.com/pets/pet/processor_spawn_expired_test.go` | Expired pet is not summonable |
| `services/atlas-inventory/atlas.com/inventory/compartment/processor_reset_pet_expiration_test.go` | Cap re-derivation, petId resolution, idempotent no-op |
| `services/atlas-channel/atlas.com/channel/socket/handler/water_of_life.go` | `WaterOfLifeHandleFunc` + target selection + rejection texts |
| `services/atlas-channel/atlas.com/channel/socket/handler/water_of_life_test.go` | Pure target-selection and item-location tests |

Modified files are named per task.

---

### Task 1: `ClassificationWaterOfLife` constant

**Files:**
- Modify: `libs/atlas-constants/item/constants.go` (cash classification block, between `ClassificationPetImprints` and `ClassificationPetSkill`)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1053`
- Test: `libs/atlas-constants/item/constants_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `item.ClassificationWaterOfLife` of type `item.Classification`, value `518`. Used by Tasks 10 and 12.

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-constants/item/constants_test.go` (create the file with `package item` if it does not exist):

```go
func TestClassificationWaterOfLife(t *testing.T) {
	if ClassificationWaterOfLife != Classification(518) {
		t.Fatalf("ClassificationWaterOfLife = %d, want 518", ClassificationWaterOfLife)
	}
	if got := GetClassification(Id(5180000)); got != ClassificationWaterOfLife {
		t.Fatalf("GetClassification(5180000) = %d, want %d", got, ClassificationWaterOfLife)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/ -run TestClassificationWaterOfLife -v`
Expected: FAIL — `undefined: ClassificationWaterOfLife`

- [ ] **Step 3: Add the constant**

In `libs/atlas-constants/item/constants.go`, insert between the `ClassificationPetImprints` and `ClassificationPetSkill` lines so the block stays numerically ordered:

```go
	ClassificationPetImprints              = Classification(517)
	ClassificationWaterOfLife              = Classification(518)
	ClassificationPetSkill                 = Classification(519)
```

Re-align the whole const block with `gofmt` (the added name is longer than its neighbours, so the alignment column moves).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./item/ -run TestClassificationWaterOfLife -v`
Expected: PASS

- [ ] **Step 5: Replace the bare literal in the channel**

`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go` around line 1053 currently reads `if category == 518 {`. Rewrite it in terms of the constant. The surrounding function is `GetCashSlotItemType`; `category` is a numeric classification. Change:

```go
		if category == 518 {
```

to:

```go
		if category == uint32(item.ClassificationWaterOfLife) {
```

Adjust the cast to match `category`'s declared type — read the surrounding declaration first and cast the constant to that type, not the variable. Add the import `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` if the file does not already have it (it may already be imported under an alias; reuse the existing alias rather than adding a second import).

The returned `CashSlotItemType(5)` is correct and must not change — it mirrors the client's `get_etc_cash_item_type` type 5.

- [ ] **Step 6: Verify the channel still builds**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: no output

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants/item/constants.go libs/atlas-constants/item/constants_test.go services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go
git commit -m "feat(task-228): add ClassificationWaterOfLife (518) and use it in GetCashSlotItemType"
```

---

### Task 2: Pets survive their own expiration (FR-1.1–1.4)

**Files:**
- Modify: `services/atlas-asset-expiration/atlas.com/asset-expiration/expiration/checker.go`
- Modify: `services/atlas-asset-expiration/atlas.com/asset-expiration/character/processor.go` (three call sites: `checkInventory`, `checkStorage`, `checkCashshop`)
- Test: `services/atlas-asset-expiration/atlas.com/asset-expiration/expiration/checker_test.go` (create)
- Test: `services/atlas-asset-expiration/atlas.com/asset-expiration/character/processor_reap_test.go` (create)

**Interfaces:**
- Consumes: `item.GetClassification`, `item.ClassificationPet` (both already exist in `libs/atlas-constants`).
- Produces: `expiration.IsReapable(templateId uint32) bool`. Not consumed by later tasks — this task is self-contained.

- [ ] **Step 1: Write the failing predicate test**

Create `services/atlas-asset-expiration/atlas.com/asset-expiration/expiration/checker_test.go`:

```go
package expiration_test

import (
	"atlas-asset-expiration/expiration"
	"testing"
)

func TestIsReapable(t *testing.T) {
	cases := []struct {
		name       string
		templateId uint32
		want       bool
	}{
		{"equip", 1002357, true},
		{"consumable", 2000000, true},
		{"etc", 4000000, true},
		{"pet (dog)", 5000000, false},
		{"pet (high id in 500)", 5009999, false},
		{"cash, not a pet (character effect)", 5010000, true},
		{"water of life", 5180000, true},
	}
	for _, c := range cases {
		if got := expiration.IsReapable(c.templateId); got != c.want {
			t.Errorf("%s: IsReapable(%d) = %v, want %v", c.name, c.templateId, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-asset-expiration/atlas.com/asset-expiration && go test ./expiration/ -run TestIsReapable -v`
Expected: FAIL — `undefined: expiration.IsReapable`

- [ ] **Step 3: Implement `IsReapable`**

Append to `services/atlas-asset-expiration/atlas.com/asset-expiration/expiration/checker.go` (and add the import `"github.com/Chronicle20/atlas/libs/atlas-constants/item"`):

```go
// IsReapable reports whether an expired asset may be destroyed. Pets are the
// sole exemption: an expired pet does not vanish, it dries up into a doll that
// keeps its cash-inventory slot until a Water of Life (5180000) revives it.
// The rule is by classification, not an id allowlist, so every present and
// future 5000xxx pet template is covered without further edits.
func IsReapable(templateId uint32) bool {
	return item.GetClassification(item.Id(templateId)) != item.ClassificationPet
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-asset-expiration/atlas.com/asset-expiration && go test ./expiration/ -run TestIsReapable -v`
Expected: PASS

- [ ] **Step 5: Write the failing three-sweep regression test**

Create `services/atlas-asset-expiration/atlas.com/asset-expiration/character/processor_reap_test.go`. It stubs the four upstream services with `httptest` and passes a recording `producer.Provider` into `CheckAndExpire`, then asserts what was emitted.

```go
package character_test

import (
	"atlas-asset-expiration/character"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// recorder captures every (topic, message) pair CheckAndExpire emits.
type recorder struct {
	mu     sync.Mutex
	topics []string
}

func (r *recorder) provider() producer.Provider {
	return func(token string) producer.MessageProducer {
		return func(p model.Provider[[]kafka.Message]) error {
			msgs, err := p()
			if err != nil {
				return err
			}
			r.mu.Lock()
			defer r.mu.Unlock()
			for range msgs {
				r.topics = append(r.topics, token)
			}
			return nil
		}
	}
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.topics)
}

const past = "2000-01-01T00:00:00Z"

// stubs serves the four upstream reads CheckAndExpire performs. Every asset it
// returns is already expired; templateId is the only variable under test.
func stubs(t *testing.T, templateId uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		path := r.URL.Path
		switch {
		case strings.Contains(path, "/compartments/") && strings.Contains(path, "/assets"):
			_, _ = w.Write([]byte(`{"data":[{"id":"1","type":"assets","attributes":{"templateId":` +
				itoa(templateId) + `,"slot":1,"expiration":"` + past + `"}}],` +
				`"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`))
		case strings.Contains(path, "/inventories"):
			_, _ = w.Write([]byte(`{"data":{"id":"1","type":"inventories","attributes":{},` +
				`"relationships":{"compartments":{"data":[{"id":"11111111-1111-1111-1111-111111111111","type":"compartments"}]}}},` +
				`"included":[{"id":"11111111-1111-1111-1111-111111111111","type":"compartments","attributes":{"type":5,"capacity":24}}]}`))
		case strings.Contains(path, "/storages"):
			_, _ = w.Write([]byte(`{"data":[{"id":"2","type":"assets","attributes":{"templateId":` +
				itoa(templateId) + `,"slot":1,"expiration":"` + past + `"}}],` +
				`"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`))
		case strings.Contains(path, "/cash-shop") || strings.Contains(path, "/cashshop"):
			_, _ = w.Write([]byte(`{"data":[{"id":"3","type":"items","attributes":{"templateId":` +
				itoa(templateId) + `,"expiration":"` + past + `"}}],` +
				`"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`))
		default:
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	t.Cleanup(srv.Close)
	for _, env := range []string{"INVENTORY_SERVICE_URL", "STORAGE_SERVICE_URL", "CASH_SHOP_SERVICE_URL", "DATA_SERVICE_URL"} {
		t.Setenv(env, srv.URL+"/")
	}
}

func itoa(v uint32) string {
	return strconvFormat(v)
}

func TestCheckAndExpireEmitsForExpiredNonPet(t *testing.T) {
	stubs(t, 2000000)
	r := &recorder{}
	l, _ := test.NewNullLogger()
	character.NewProcessor(l, context.Background()).CheckAndExpire(r.provider())(42, 7, 0)
	if r.count() == 0 {
		t.Fatal("expected expire commands for an expired consumable, got none")
	}
}

func TestCheckAndExpireEmitsNothingForExpiredPet(t *testing.T) {
	stubs(t, 5000000)
	r := &recorder{}
	l, _ := test.NewNullLogger()
	character.NewProcessor(l, context.Background()).CheckAndExpire(r.provider())(42, 7, 0)
	if r.count() != 0 {
		t.Fatalf("expected no expire commands for an expired pet, got %d on topics %v", r.count(), r.topics)
	}
}
```

Add `strconvFormat` as a tiny local helper at the bottom of the file:

```go
func strconvFormat(v uint32) string {
	return strconv.FormatUint(uint64(v), 10)
}
```

and import `"strconv"`.

Before running, read `services/atlas-asset-expiration/atlas.com/asset-expiration/{inventory,storage,cashshop}/requests.go` and confirm the exact env-var names (`requests.RootUrl("…")`) and URL paths, then correct the `switch` arms and the `t.Setenv` list to match. The stub must serve every URL those three clients actually request — if a path falls through to the `default` arm the sweep silently finds nothing and *both* tests pass for the wrong reason. Assert this by checking that `TestCheckAndExpireEmitsForExpiredNonPet` fails when you comment out its `stubs` call.

- [ ] **Step 6: Run tests to verify the pet case fails**

Run: `cd services/atlas-asset-expiration/atlas.com/asset-expiration && go test ./character/ -run TestCheckAndExpire -v`
Expected: `TestCheckAndExpireEmitsForExpiredNonPet` PASS, `TestCheckAndExpireEmitsNothingForExpiredPet` FAIL (three expire commands emitted)

- [ ] **Step 7: Apply the predicate at all three call sites**

In `services/atlas-asset-expiration/atlas.com/asset-expiration/character/processor.go`:

`checkInventory` — change

```go
			if expiration.IsExpired(a.Expiration, now) {
```

to

```go
			if expiration.IsExpired(a.Expiration, now) && expiration.IsReapable(a.TemplateId) {
```

`checkStorage` — change

```go
		if expiration.IsExpired(a.Expiration, now) {
```

to

```go
		if expiration.IsExpired(a.Expiration, now) && expiration.IsReapable(a.TemplateId) {
```

`checkCashshop` — change

```go
		if expiration.IsExpired(item.Expiration, now) {
```

to

```go
		if expiration.IsExpired(item.Expiration, now) && expiration.IsReapable(item.TemplateId) {
```

Note the cash-shop loop variable is named `item`, which shadows nothing here (this file does not import `libs/atlas-constants/item`); leave the name alone.

Nothing else in the service changes, so FR-1.3 holds by construction.

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd services/atlas-asset-expiration/atlas.com/asset-expiration && go test -race ./... -v`
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add services/atlas-asset-expiration/atlas.com/asset-expiration/expiration/ services/atlas-asset-expiration/atlas.com/asset-expiration/character/
git commit -m "feat(task-228): exempt pets from the expiration sweep so they dry up into dolls"
```

---

### Task 3: `atlas-data` parses `info/life` for cash items (FR-8.1, FR-8.2)

**Files:**
- Modify: `services/atlas-data/atlas.com/data/cash/reader.go` (beside the `MaxDays` read, ~line 80)
- Modify: `services/atlas-data/atlas.com/data/cash/rest.go` (`RestModel`)
- Test: `services/atlas-data/atlas.com/data/cash/reader_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `cash.RestModel.Life uint32` serialized as `"life"` with `omitempty`, in **days**. Consumed over REST by Tasks 8 (atlas-pets), 9 (atlas-inventory) and 12 (atlas-channel).

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-data/atlas.com/data/cash/reader_test.go`. Read the existing tests in that file first and reuse their fixture-construction idiom verbatim (the XML node type and the reader entry point); the assertions below are what must hold:

```go
func TestReaderParsesLife(t *testing.T) {
	// 0518.img/05180000/info: slotMax=1, cash=1, life=90. No maxDays, no addTime.
	models := readCashFixture(t, `
		<imgdir name="0518.img">
			<imgdir name="05180000">
				<imgdir name="info">
					<int name="slotMax" value="1"/>
					<int name="cash" value="1"/>
					<int name="life" value="90"/>
				</imgdir>
			</imgdir>
		</imgdir>`)

	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}
	if models[0].Life != 90 {
		t.Errorf("Life = %d, want 90", models[0].Life)
	}
	if models[0].MaxDays != 0 {
		t.Errorf("MaxDays = %d, want 0 (0518.img has no maxDays node)", models[0].MaxDays)
	}
}

func TestReaderLifeAbsentIsZeroAndOmitted(t *testing.T) {
	models := readCashFixture(t, `
		<imgdir name="0518.img">
			<imgdir name="05180000">
				<imgdir name="info">
					<int name="slotMax" value="1"/>
				</imgdir>
			</imgdir>
		</imgdir>`)

	if models[0].Life != 0 {
		t.Errorf("Life = %d, want 0", models[0].Life)
	}
	b, err := json.Marshal(models[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"life"`) {
		t.Errorf("absent life must be omitted from JSON, got %s", b)
	}
}
```

`readCashFixture` is whatever the existing tests in the file already use to turn fixture XML into `[]RestModel`; if the file has no such helper, write one modelled on the existing test's setup rather than inventing a new reader entry point.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-data/atlas.com/data && go test ./cash/ -run TestReaderParsesLife -v`
Expected: FAIL — `models[0].Life undefined`

- [ ] **Step 3: Add the field and the read**

In `services/atlas-data/atlas.com/data/cash/rest.go`, add to `RestModel` directly below `MaxDays`:

```go
	// Life is info/life in DAYS — the lifespan a pet-revival item (Water of
	// Life, classification 518) grants. Same WZ node and same unit as
	// pet/reader.go's Life; 0518.img carries life but no maxDays, which is
	// why the expiration-extender's maxDays ceiling cannot serve this flow.
	Life uint32 `json:"life,omitempty"`
```

In `services/atlas-data/atlas.com/data/cash/reader.go`, directly below the `m.MaxDays = …` line:

```go
			m.Life = uint32(i.GetIntegerWithDefault("life", 0))
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-data/atlas.com/data && go test -race ./cash/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data/atlas.com/data/cash/
git commit -m "feat(task-228): parse info/life (days) on cash items and expose it on the REST model"
```

---

### Task 4: The `WATER_OF_LIFE` serverbound codec (FR-2)

**Files:**
- Create: `libs/atlas-packet/pet/serverbound/water_of_life.go`
- Create: `libs/atlas-packet/pet/serverbound/water_of_life_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `serverbound.WaterOfLifeHandle` (string constant `"WaterOfLifeHandle"`) and `serverbound.WaterOfLife` (empty struct with `Operation()`, `String()`, `Encode`, `Decode`). Consumed by Task 5 (template `handler` value) and Task 12 (channel `handlerMap` key + decode).

The family is `pet/serverbound` — matching `PET_AUTO_POT` (`pet/serverbound/PetItemUse`, fname `CWvsContext::SendStatChangeItemUseRequestByPetQ`), which shows the matrix groups by feature, not by client class.

**There are no version gates.** The struct has zero fields, so there is nothing to gate; `docs/packets/gates.yaml` is a wire-*divergence* registry and gets no entry. Version scope is enforced entirely by Task 5's template routing and by the matrix's `n-a` cells.

- [ ] **Step 1: Write the failing round-trip test**

Create `libs/atlas-packet/pet/serverbound/water_of_life_test.go`. Read `libs/atlas-packet/character/serverbound/chalkboard_close_test.go` first and mirror its structure exactly — it is the repo's only other empty-body serverbound codec and its test already solves the tenant/reader plumbing:

```go
package serverbound

// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v83 ida=0xa1dce6
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v84 ida=0xa68f85
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v87 ida=0xab501c
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v92 ida=0x9c6f90
// packet-audit:verify packet=pet/serverbound/WaterOfLife version=gms_v95 ida=0x9f28e0
//
// CWvsContext::SendWaterOfLife constructs COutPacket(op) and sends it with no
// Encode calls at all on every applicable version, so the body is zero bytes.
```

plus a test body that, for each of the five version keys, encodes a `WaterOfLife{}`, asserts the result is a zero-length byte slice, decodes those bytes back into a fresh `WaterOfLife{}`, and asserts the round trip leaves the struct equal to the original. Use the same per-version tenant construction the chalkboard test uses.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-packet && go test ./pet/serverbound/ -run WaterOfLife -v`
Expected: FAIL — `undefined: WaterOfLife`

- [ ] **Step 3: Write the codec**

Create `libs/atlas-packet/pet/serverbound/water_of_life.go`:

```go
package serverbound

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

const WaterOfLifeHandle = "WaterOfLifeHandle"

// WaterOfLife - CWvsContext::SendWaterOfLife
//
// The body is empty on every applicable version. The client reaches this from
// CWvsContext::SendEtcCashItemUseRequest (gms_v83 @0xa1dc5b), which switches on
// get_etc_cash_item_type (@0x486845) and, for classification 518 => type 5,
// calls SendWaterOfLife() -- a distinct opcode with no body, NOT a CASH_ITEM_USE
// sub-body. Verified per IDB: v83 @0xa1dce6, v84 @0xa68f85, v87 @0xab501c,
// v92 @0x9c6f90, v95 @0x9f28e0 -- each is COutPacket(op) + SendPacket with zero
// Encode* calls. No field diverges, so there are no version gates.
//
// Every operand is derived server-side: the target pet and the consumed Water
// of Life are resolved by atlas-channel, not named on the wire.
type WaterOfLife struct{}

func (m WaterOfLife) Operation() string {
	return WaterOfLifeHandle
}

func (m WaterOfLife) String() string {
	return ""
}

func (m WaterOfLife) Encode(_ logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	return func(options map[string]interface{}) []byte {
		return []byte{}
	}
}

func (m *WaterOfLife) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-packet && go test -race ./pet/serverbound/ -run WaterOfLife -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/pet/serverbound/water_of_life.go libs/atlas-packet/pet/serverbound/water_of_life_test.go
git commit -m "feat(task-228): add the empty-body WATER_OF_LIFE serverbound codec"
```

---

### Task 5: Route the handler in the five applicable seed templates (FR-3.2–3.4)

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_92_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`

**Interfaces:**
- Consumes: `WaterOfLifeHandle` from Task 4 (the `handler` string must match the codec's `Operation()` return value exactly, or the dispatch map never binds).
- Produces: a bound opcode per version. Consumed at runtime by Task 12's handler.

Insertion positions were computed from the current templates; in every case the entry goes **immediately after the `SueCharacter` entry**:

| Template | `opCode` | Previous entry | Next entry |
|---|---|---|---|
| `template_gms_83_1.json` | `"0x75"` | `0x72` `SueCharacter` | `0x76` `AdminChat` |
| `template_gms_84_1.json` | `"0x75"` | `0x72` `SueCharacter` | `0x78` `AdminChat` |
| `template_gms_87_1.json` | `"0x78"` | `0x75` `SueCharacter` | `0x7C` `AdminChat` |
| `template_gms_92_1.json` | `"0x80"` | `0x7D` `SueCharacter` | `0x8C` `MessengerOperationHandle` |
| `template_gms_95_1.json` | `"0x81"` | `0x7E` `SueCharacter` | `0x8B` `AdminChat` |

- [ ] **Step 1: Insert the entry in all five templates**

In each file, inside `socket.handlers`, immediately after the `SueCharacter` object, insert (substituting the per-version `opCode` from the table):

```json
      {
        "opCode": "0x75",
        "validator": "LoggedInValidator",
        "handler": "WaterOfLifeHandle",
        "fname": "CWvsContext::SendWaterOfLife",
        "services": ["channel"]
      },
```

Match the surrounding file's indentation and key order exactly. Use `Edit` per file — do **not** use a shell patch loop.

Do not touch `template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json` or `template_jms_185_1.json`.

- [ ] **Step 2: Verify the JSON still parses and the opcodes landed where intended**

Run:

```bash
python3 - <<'EOF'
import json
def num(c):
    s=str(c); return int(s,16) if s.lower().startswith("0x") else int(s)
for v,op in [("83",0x75),("84",0x75),("87",0x78),("92",0x80),("95",0x81)]:
    p=f"services/atlas-configurations/seed-data/templates/template_gms_{v}_1.json"
    hs=json.load(open(p))["socket"]["handlers"]
    hit=[h for h in hs if h.get("handler")=="WaterOfLifeHandle"]
    codes=[num(h["opCode"]) for h in hs]
    print(v, "entry:", hit, "sorted:", codes==sorted(codes))
for v in ["12","48","61","72","79"]:
    hs=json.load(open(f"services/atlas-configurations/seed-data/templates/template_gms_{v}_1.json"))["socket"]["handlers"]
    print(v, "absent:", not any(h.get("handler")=="WaterOfLifeHandle" for h in hs))
hs=json.load(open("services/atlas-configurations/seed-data/templates/template_jms_185_1.json"))["socket"]["handlers"]
print("jms absent:", not any(h.get("handler")=="WaterOfLifeHandle" for h in hs))
EOF
```

Expected: one entry per applicable version with the correct `opCode`, `sorted: True` for all five, `absent: True` for all six non-applicable templates.

- [ ] **Step 3: Run the template guards**

Run from the worktree root:

```bash
tools/template-opcode-order-guard.sh && tools/template-duplicate-binding-guard.sh && tools/template-movement-types-guard.sh
```

Expected: all exit 0.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "feat(task-228): route WaterOfLifeHandle in the five applicable seed templates"
```

---

### Task 6: `libs/atlas-saga` — the `revive_pet` action (design §7)

**Files:**
- Modify: `libs/atlas-saga/model.go` (saga `Type` block ~line 30-50; `Action` block ~line 98)
- Modify: `libs/atlas-saga/payloads.go` (beside `EvolvePetPayload`, ~line 291)
- Modify: `libs/atlas-saga/unmarshal.go` (beside the `case EvolvePet:` arm, ~line 186)
- Test: `libs/atlas-saga/unmarshal_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `saga.PetRevive` of type `Type`, value `"pet_revive"`
  - `saga.RevivePet` of type `Action`, value `"revive_pet"`
  - `saga.RevivePetPayload{CharacterId uint32 \`json:"characterId"\`; PetId uint32 \`json:"petId"\`; SourceTemplateId uint32 \`json:"sourceTemplateId"\`}`

  Consumed by Tasks 11 (orchestrator) and 12 (channel).

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-saga/unmarshal_test.go`:

```go
func TestPetReviveSagaTypeValue(t *testing.T) {
	if PetRevive != Type("pet_revive") {
		t.Fatalf("PetRevive = %q, want %q", PetRevive, "pet_revive")
	}
}

func TestRevivePetActionValue(t *testing.T) {
	if RevivePet != Action("revive_pet") {
		t.Fatalf("RevivePet = %q, want %q", RevivePet, "revive_pet")
	}
}

func TestUnmarshalRevivePetPayload(t *testing.T) {
	raw := []byte(`{"stepId":"revive_pet","status":"pending","action":"revive_pet",` +
		`"payload":{"characterId":42,"petId":7,"sourceTemplateId":5180000}}`)

	var s Step[any]
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p, ok := s.Payload.(RevivePetPayload)
	if !ok {
		t.Fatalf("payload type = %T, want RevivePetPayload", s.Payload)
	}
	want := RevivePetPayload{CharacterId: 42, PetId: 7, SourceTemplateId: 5180000}
	if p != want {
		t.Fatalf("payload = %+v, want %+v", p, want)
	}
}
```

Match the existing tests in that file for how `Step[any]` is unmarshalled (field names on the JSON envelope) — read one of them first and copy its envelope shape rather than trusting the sketch above.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-saga && go test . -run 'PetRevive|RevivePet' -v`
Expected: FAIL — `undefined: PetRevive`

- [ ] **Step 3: Add the type, action, payload and unmarshal arm**

`libs/atlas-saga/model.go` — in the `Type` const block, after `RemoteMerchant`:

```go
	// PetRevive is the classification-518 Water of Life flow: consume the item,
	// then reset a dried-up pet's lifespan. Consume comes first so a failed
	// revive compensates into a refund rather than a free revive (task-228).
	PetRevive Type = "pet_revive"
```

`libs/atlas-saga/model.go` — in the `Action` const block, directly after `EvolvePet`:

```go
	RevivePet              Action = "revive_pet"
```

`libs/atlas-saga/payloads.go` — directly after `EvolvePetPayload`:

```go
// RevivePetPayload drives a Water of Life pet revive. It deliberately carries
// NO expiration: atlas-pets derives the new lifespan from the consumed item's
// own WZ info/life, so a forged saga step cannot dictate one. SourceTemplateId
// names the consumed Water of Life (classification 518).
type RevivePetPayload struct {
	CharacterId      uint32 `json:"characterId"`
	PetId            uint32 `json:"petId"`
	SourceTemplateId uint32 `json:"sourceTemplateId"`
}
```

`libs/atlas-saga/unmarshal.go` — directly after the `case EvolvePet:` arm:

```go
	case RevivePet:
		var payload RevivePetPayload
		if err := json.Unmarshal(aux.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal payload for action %s: %w", s.Action, err)
		}
		s.Payload = any(payload).(T)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-saga && go test -race . -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-saga/
git commit -m "feat(task-228): add the revive_pet saga action, payload and pet_revive saga type"
```

---

### Task 7: An expired pet is not summonable (FR-1.5, design §3)

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/processor.go` (`Spawn`, ~line 422; error vars ~line 418)
- Test: `services/atlas-pets/atlas.com/pets/pet/processor_spawn_expired_test.go` (create)

**Interfaces:**
- Consumes: nothing.
- Produces: `pet.ErrPetExpired`. Not consumed by later tasks.

This is deliberately the *only* gate: after Task 2 nothing in Atlas observes the moment a pet expires, and adding an expiry watcher purely to force-despawn a running pet would reintroduce the per-minute pet scan Task 2 just removed, to serve a case the client has no concept of. `pet.DespawnReasonExpired` stays unused; do not start emitting it.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-pets/atlas.com/pets/pet/processor_spawn_expired_test.go`. Read the existing tests under `services/atlas-pets/atlas.com/pets/pet/` first and reuse their database/fixture setup verbatim. The behaviour to assert:

- A pet whose `Expiration()` is in the past: `Spawn(mb)(petId)(ownerId)(false)` returns an error matching `errors.Is(err, pet.ErrPetExpired)`, and the pet's `Slot()` is unchanged (still `-1`).
- A pet whose `Expiration()` is in the future spawns normally (slot becomes `>= 0`) — the regression guard proving the gate does not fire on live pets.
- A pet whose `Expiration()` is the zero time spawns normally — a permanent pet is not "expired".

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run Expired -v`
Expected: FAIL — `undefined: pet.ErrPetExpired`

- [ ] **Step 3: Add the gate**

In `services/atlas-pets/atlas.com/pets/pet/processor.go`, extend the existing error block:

```go
var (
	ErrTooManySpawnedPets = errors.New("too many pets spawned")
	ErrNeedMultiPetSkill  = errors.New("need multi pet skill")
	// ErrPetExpired: a pet whose lifespan has run out is a dried-up doll. It
	// keeps its inventory slot (see atlas-asset-expiration's IsReapable) but
	// cannot be summoned until a Water of Life revives it (task-228).
	ErrPetExpired = errors.New("pet expired")
)
```

In `Spawn`, inside the transaction, directly after the existing ownership check (`if pe.OwnerId() != actorId { … }`) and **before** the egg hatch-on-summon branch:

```go
					if !pe.Expiration().IsZero() && time.Now().After(pe.Expiration()) {
						return ErrPetExpired
					}
```

Placing it before the egg branch matters: an expired egg must not hatch either, and hatching mutates the row.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-pets/atlas.com/pets && go test -race ./pet/ -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/pet/
git commit -m "feat(task-228): refuse to summon an expired pet"
```

---

### Task 8: `atlas-pets` gains a cash-data client and the revive contracts

**Files:**
- Create: `services/atlas-pets/atlas.com/pets/data/cash/model.go`
- Create: `services/atlas-pets/atlas.com/pets/data/cash/builder.go`
- Create: `services/atlas-pets/atlas.com/pets/data/cash/rest.go`
- Create: `services/atlas-pets/atlas.com/pets/data/cash/requests.go`
- Create: `services/atlas-pets/atlas.com/pets/data/cash/processor.go`
- Create: `services/atlas-pets/atlas.com/pets/data/cash/mock/processor.go`
- Modify: `services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go`
- Modify: `services/atlas-pets/atlas.com/pets/kafka/message/compartment/kafka.go`
- Modify: `services/atlas-pets/atlas.com/pets/inventory/command.go`
- Test: `services/atlas-pets/atlas.com/pets/data/cash/processor_test.go` (create)

**Interfaces:**
- Consumes: `cash.RestModel.Life` from Task 3 (over REST, `GET {DATA}/data/cash/{itemId}`).
- Produces, all consumed by Task 9:
  - `cash.Processor` interface with `GetById(itemId uint32) (Model, error)`; `cash.NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`
  - `cash.Model` with `Id() uint32` and `Life() uint32`; `cash.NewModelBuilder(id uint32) *ModelBuilder` with `SetLife(uint32) *ModelBuilder` and `Build() Model`
  - `petmsg.CommandPetRevive = "REVIVE"`, `petmsg.ReviveCommandBody{SourceTemplateId uint32}`
  - `petmsg.StatusEventTypeRevived = "REVIVED"`, `petmsg.RevivedStatusEventBody{Slot int8; Expiration time.Time; TransactionId uuid.UUID}`
  - `petmsg.StatusEventTypeReviveFailed = "REVIVE_FAILED"`, `petmsg.ReviveFailedStatusEventBody{Reason string; TransactionId uuid.UUID}`
  - `compartmentmsg.CommandResetPetExpiration = "RESET_PET_EXPIRATION"`, `compartmentmsg.ResetPetExpirationCommandBody{PetId uint32; Expiration time.Time; SourceTemplateId uint32}`
  - `inventory.ProcessorImpl.ResetPetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error`

  Task 11 (orchestrator) mirrors `CommandPetRevive` / `ReviveCommandBody` / the two status events in its own module; Task 9 (atlas-inventory) mirrors `CommandResetPetExpiration` / `ResetPetExpirationCommandBody`.

- [ ] **Step 1: Write the failing cash-client test**

Create `services/atlas-pets/atlas.com/pets/data/cash/processor_test.go`, modelled on `services/atlas-inventory/atlas.com/inventory/data/cash/processor_test.go` (read it first and copy its httptest + `t.Setenv("DATA_SERVICE_URL", …)` idiom):

```go
package cash_test

import (
	"atlas-pets/data/cash"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

func TestGetByIdReadsLife(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"id":"5180000","type":"cash_items","attributes":{"life":90}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	m, err := cash.NewProcessor(l, context.Background()).GetById(5180000)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.Life() != 90 {
		t.Errorf("Life = %d, want 90", m.Life())
	}
}

func TestGetByIdLifeAbsentIsZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"id":"5180000","type":"cash_items","attributes":{}}}`))
	}))
	defer srv.Close()
	t.Setenv("DATA_SERVICE_URL", srv.URL+"/")

	l, _ := test.NewNullLogger()
	m, err := cash.NewProcessor(l, context.Background()).GetById(5180000)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if m.Life() != 0 {
		t.Errorf("Life = %d, want 0", m.Life())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./data/cash/ -v`
Expected: FAIL — package `atlas-pets/data/cash` does not exist

- [ ] **Step 3: Write the cash-data client**

`services/atlas-pets/atlas.com/pets/data/cash/model.go`:

```go
package cash

// Model is a cash item template's pet-revival attributes, as resolved from
// atlas-data.
type Model struct {
	id   uint32
	life uint32
}

func (m Model) Id() uint32 { return m.id }

// Life is info/life in DAYS — the lifespan a Water of Life grants a revived
// pet. Zero means the WZ node is absent, which the revive treats as a data
// error (reject, consume nothing).
func (m Model) Life() uint32 { return m.life }
```

`services/atlas-pets/atlas.com/pets/data/cash/builder.go`:

```go
package cash

// ModelBuilder constructs a Model. It exists so callers outside this package
// (notably test doubles) can build one without exported fields, per the
// project's Builder convention.
type ModelBuilder struct {
	id   uint32
	life uint32
}

func NewModelBuilder(id uint32) *ModelBuilder {
	return &ModelBuilder{id: id}
}

func (b *ModelBuilder) SetLife(v uint32) *ModelBuilder { b.life = v; return b }

func (b *ModelBuilder) Build() Model {
	return Model{id: b.id, life: b.life}
}
```

`services/atlas-pets/atlas.com/pets/data/cash/rest.go`:

```go
package cash

import (
	"strconv"
)

// RestModel is the subset of atlas-data's cash resource this service needs:
// info/life, used to derive a revived pet's new lifespan server-side.
type RestModel struct {
	Id   uint32 `json:"-"`
	Life uint32 `json:"life,omitempty"`
}

func (r RestModel) GetName() string {
	return "cash_items"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

// SetToOneReferenceID / SetToManyReferenceIDs are required by api2go's
// unmarshal whenever the upstream response carries a relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{id: rm.Id, life: rm.Life}, nil
}
```

`services/atlas-pets/atlas.com/pets/data/cash/requests.go`:

```go
package cash

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "data/cash"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, itemId))
}
```

Before running the test, confirm the resource path against `services/atlas-inventory/atlas.com/inventory/data/cash/requests.go` and use whatever that file uses — it is the working client for the same endpoint.

`services/atlas-pets/atlas.com/pets/data/cash/processor.go`:

```go
package cash

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetById(itemId uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetById(itemId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(itemId), Extract)()
}
```

`services/atlas-pets/atlas.com/pets/data/cash/mock/processor.go` — mirror `services/atlas-pets/atlas.com/pets/data/pet/mock/` exactly in style:

```go
package mock

import (
	"atlas-pets/data/cash"
)

// ProcessorMock is a function-field stub of cash.Processor.
type ProcessorMock struct {
	GetByIdFunc func(itemId uint32) (cash.Model, error)
}

var _ cash.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetById(itemId uint32) (cash.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(itemId)
	}
	return cash.Model{}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd services/atlas-pets/atlas.com/pets && go test -race ./data/cash/ -v`
Expected: PASS

- [ ] **Step 5: Add the pet command and status-event contracts**

In `services/atlas-pets/atlas.com/pets/kafka/message/pet/kafka.go`:

Add to the command const block, after `CommandSetSkill`:

```go
	CommandPetRevive         = "REVIVE"
```

Add the body, after `EvolveCommandBody`:

```go
// ReviveCommandBody restores a dried-up pet's lifespan. It carries NO
// expiration: atlas-pets derives it from the consumed item's own WZ data, so a
// forged command cannot dictate a lifespan. SourceTemplateId names the consumed
// Water of Life (classification 518). Command[E] already carries TransactionId,
// ActorId and PetId, so the body needs nothing else.
type ReviveCommandBody struct {
	SourceTemplateId uint32 `json:"sourceTemplateId"`
}
```

Add to the status-event const block, after `StatusEventTypeFlagChanged`:

```go
	StatusEventTypeRevived      = "REVIVED"
	StatusEventTypeReviveFailed = "REVIVE_FAILED"
```

Add the bodies at the end of the file, after `EvolvedStatusEventBody`:

```go
// RevivedStatusEventBody reports a successful Water of Life revive. Expiration
// is the absolute new dry-up instant; Slot is unchanged by the revive (a doll
// stays unsummoned) and is carried only so consumers need no extra read.
type RevivedStatusEventBody struct {
	Slot          int8      `json:"slot"`
	Expiration    time.Time `json:"expiration"`
	TransactionId uuid.UUID `json:"transactionId"`
}

// ReviveFailedStatusEventBody is a REAL terminal failure event, not a silent
// drop. By the time REVIVE runs the player's Water of Life is already
// destroyed by the saga's first step, so a timeout-length wait for the refund
// would read as a lost item; the saga accepts this event and compensates
// immediately.
type ReviveFailedStatusEventBody struct {
	Reason        string    `json:"reason"`
	TransactionId uuid.UUID `json:"transactionId"`
}
```

Add `"time"` to the file's imports.

- [ ] **Step 6: Add the compartment mirror and the cascade buffer function**

In `services/atlas-pets/atlas.com/pets/kafka/message/compartment/kafka.go`:

```go
const (
	EnvCommandTopic           = "COMMAND_TOPIC_COMPARTMENT"
	CommandChangeTemplate     = "CHANGE_TEMPLATE"
	CommandResetPetExpiration = "RESET_PET_EXPIRATION"
)
```

and add the body (add the `"time"` import):

```go
// ResetPetExpirationCommandBody sets a dried-up pet asset's expiration to an
// absolute instant. The asset is resolved by (CharacterId, PetId) — never by
// slot — mirroring ChangeTemplateCommandBody. SourceTemplateId names the
// consumed Water of Life so atlas-inventory can re-derive the ceiling itself;
// the sender is not a trust boundary. Absolute (not a duration) so a
// redelivered command is a no-op rather than a second grant.
//
// MIRROR: this struct is duplicated in
// services/atlas-inventory/.../kafka/message/compartment/kafka.go. The two live
// in separate Go modules, so a field name or json tag changed in one and not
// the other fails no build — it decodes into a zero-valued body at runtime.
type ResetPetExpirationCommandBody struct {
	PetId            uint32    `json:"petId"`
	Expiration       time.Time `json:"expiration"`
	SourceTemplateId uint32    `json:"sourceTemplateId"`
}
```

In `services/atlas-pets/atlas.com/pets/inventory/command.go`, append (mirroring `ChangeTemplate` exactly):

```go
// ResetPetExpiration buffers a RESET_PET_EXPIRATION command to atlas-inventory.
// It is buffered inside the revive's own database transaction + outbox, so the
// pet row update and this cascade commit together or not at all — the pet
// record and the inventory slot cannot diverge (design §7.1).
func (p *ProcessorImpl) ResetPetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error {
		return mb.Put(compartmentmsg.EnvCommandTopic, resetPetExpirationCommandProvider(transactionId, characterId, petId, expiration, sourceTemplateId))
	}
}

func resetPetExpirationCommandProvider(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &compartmentmsg.Command[compartmentmsg.ResetPetExpirationCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueCash),
		Type:          compartmentmsg.CommandResetPetExpiration,
		Body: compartmentmsg.ResetPetExpirationCommandBody{
			PetId:            petId,
			Expiration:       expiration,
			SourceTemplateId: sourceTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Add `"time"` to that file's imports. If `inventory.Processor` is an interface in this package, add `ResetPetExpiration` to it with the same signature.

- [ ] **Step 7: Verify the module builds and tests stay green**

Run: `cd services/atlas-pets/atlas.com/pets && go build ./... && go test -race ./... `
Expected: build clean, all tests PASS

- [ ] **Step 8: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/data/cash/ services/atlas-pets/atlas.com/pets/kafka/message/ services/atlas-pets/atlas.com/pets/inventory/command.go
git commit -m "feat(task-228): add the atlas-pets cash-data client and the REVIVE / RESET_PET_EXPIRATION contracts"
```

---

### Task 9: `atlas-pets` — the `REVIVE` command (design §8.1, §9)

**Files:**
- Modify: `services/atlas-pets/atlas.com/pets/pet/entity.go` (`Entity`, `Make`)
- Modify: `services/atlas-pets/atlas.com/pets/pet/model.go` (`Model` + getter)
- Modify: `services/atlas-pets/atlas.com/pets/pet/builder.go` (`ModelBuilder`, `Clone`, setter)
- Modify: `services/atlas-pets/atlas.com/pets/pet/administrator.go` (new `updateOnRevive`)
- Modify: `services/atlas-pets/atlas.com/pets/pet/processor.go` (new `Revive` / `ReviveAndEmit`, event providers)
- Modify: `services/atlas-pets/atlas.com/pets/kafka/consumer/pet/consumer.go` (new handler arm)
- Test: `services/atlas-pets/atlas.com/pets/pet/processor_revive_test.go` (create)

**Interfaces:**
- Consumes: Task 8's `cash.Processor`, `petmsg.CommandPetRevive`, `petmsg.ReviveCommandBody`, `petmsg.StatusEventTypeRevived`, `petmsg.StatusEventTypeReviveFailed`, `petmsg.RevivedStatusEventBody`, `petmsg.ReviveFailedStatusEventBody`, `inventory.ProcessorImpl.ResetPetExpiration`.
- Produces:
  - `pet.Model.ReviveTransactionId() *uuid.UUID`, `pet.ModelBuilder.SetReviveTransactionId(*uuid.UUID)`
  - `pet.ProcessorImpl.Revive(mb *message.Buffer) func(transactionId uuid.UUID, actorId uint32, petId uint32, sourceTemplateId uint32) error`
  - `pet.ProcessorImpl.ReviveAndEmit(transactionId uuid.UUID, actorId uint32, petId uint32, sourceTemplateId uint32) error`

  `ReviveAndEmit` is consumed by the consumer arm in this task only; the orchestrator (Task 11) reaches it over Kafka.

- [ ] **Step 1: Write the failing revive tests**

Create `services/atlas-pets/atlas.com/pets/pet/processor_revive_test.go`. Reuse the database/fixture setup from the existing tests in that directory and inject the cash client with `data/cash/mock`. Assert:

1. **Happy path.** Given a pet with `Expiration()` one hour in the past, name `"Mochi"`, level 7, closeness 300, fullness 42, slot `-1`, and a cash mock returning `Life() == 90`: after `ReviveAndEmit(tx, ownerId, petId, 5180000)` the persisted pet's expiration is within one minute of `time.Now().Add(90 * 24 * time.Hour)`, and name / level / closeness / fullness / slot / templateId / flag are byte-for-byte unchanged.
2. **Lifespan comes from data, not a constant.** The same fixture with a cash mock returning `Life() == 30` produces an expiration within one minute of `now + 30 days`. This is the FR-5.1 "not a constant" guard — the numbers must differ between cases 1 and 2 or the test proves nothing.
3. **Zero `life` is a data error.** Cash mock returning `Life() == 0`: the pet row is unchanged (expiration still in the past, `revive_transaction_id` still `nil`) and a `REVIVE_FAILED` message is present on `EnvStatusEventTopic` in the emitted buffer.
4. **Cascade.** On the happy path a `RESET_PET_EXPIRATION` command is present on `EnvCommandTopic` (the compartment topic) carrying the *same* expiration the pet row now holds, `PetId` = the pet, and `SourceTemplateId` = 5180000.
5. **Idempotency row A — redelivery.** A pet already revived by transaction `T` (its `revive_transaction_id == T`, expiration in the future): re-running `Revive` with the same `T` performs **no** write (expiration byte-identical), re-buffers `RESET_PET_EXPIRATION` with the **stored** expiration, and re-emits `REVIVED`.
6. **Idempotency row B — live pet, different transaction.** Same pet, different transaction id: `REVIVE_FAILED` is emitted, no write occurs.
7. **Idempotency row C — expired pet.** `revive_transaction_id` `nil`, expiration in the past: proceeds (covered by case 1).
8. **Ownership.** `ActorId` not equal to the pet's `OwnerId()`: `REVIVE_FAILED`, no write.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-pets/atlas.com/pets && go test ./pet/ -run Revive -v`
Expected: FAIL — `undefined: ReviveAndEmit`

- [ ] **Step 3: Add the persistence column**

`services/atlas-pets/atlas.com/pets/pet/entity.go` — add to `Entity` after `PurchaseBy`:

```go
	// ReviveTransactionId is the saga transaction of the last successful Water
	// of Life revive. It is what distinguishes a Kafka REDELIVERY (same id =>
	// replay, no second grant) from a genuine SECOND water used on an
	// already-revived pet (different id => reject and refund). Neither
	// "reject if alive" nor "no-op if alive" alone can tell those apart.
	ReviveTransactionId *uuid.UUID `gorm:"type:uuid"`
```

`Entity` already migrates via `pet.Migration`'s `AutoMigrate`, so no hand-written migration is needed.

Extend `Make` to carry it through:

```go
	return NewModelBuilder(e.Id, e.CashId, e.TemplateId, e.Name, e.OwnerId).
		SetLevel(e.Level).
		SetCloseness(e.Closeness).
		SetFullness(e.Fullness).
		SetExpiration(e.Expiration).
		SetSlot(*e.Slot).
		SetExcludes(es).
		SetFlag(e.Flag).
		SetPurchaseBy(e.PurchaseBy).
		SetReviveTransactionId(e.ReviveTransactionId).
		Build()
```

`services/atlas-pets/atlas.com/pets/pet/model.go` — add the field and getter:

```go
	reviveTransactionId *uuid.UUID
```

```go
// ReviveTransactionId is the saga transaction of the last successful revive,
// or nil if the pet has never been revived.
func (m Model) ReviveTransactionId() *uuid.UUID {
	return m.reviveTransactionId
}
```

`services/atlas-pets/atlas.com/pets/pet/builder.go` — add the field to `ModelBuilder`, the setter, the `Clone` chain link, and the `Build` assignment:

```go
func (b *ModelBuilder) SetReviveTransactionId(id *uuid.UUID) *ModelBuilder {
	b.reviveTransactionId = id
	return b
}
```

Add `.SetReviveTransactionId(m.ReviveTransactionId())` to `Clone`'s chain and `reviveTransactionId: b.reviveTransactionId` to `Build`'s struct literal. Import `"github.com/google/uuid"` in `model.go` and `builder.go`.

- [ ] **Step 4: Add the narrow administrator function**

`services/atlas-pets/atlas.com/pets/pet/administrator.go`, after `updateOnEvolve`:

```go
// updateOnRevive writes ONLY the expiration and the revive transaction id.
// Deliberately not updateOnEvolve: that function also rewrites template_id,
// and a revive must never touch the pet's template.
func updateOnRevive(db *gorm.DB) func(petId uint32, expiration time.Time, transactionId uuid.UUID) error {
	return func(petId uint32, expiration time.Time, transactionId uuid.UUID) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Updates(map[string]interface{}{
				"expiration":            expiration,
				"revive_transaction_id": transactionId,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("no entity found to revive")
		}
		return nil
	}
}
```

Add `"github.com/google/uuid"` to that file's imports.

- [ ] **Step 5: Add the event providers**

In `services/atlas-pets/atlas.com/pets/pet/processor.go`, beside the existing `evolvedEventProvider` (copy its exact provider signature and key-construction idiom — read it first):

```go
func revivedEventProvider(m Model, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	// mirror evolvedEventProvider's key construction
	value := &pet.StatusEvent[pet.RevivedStatusEventBody]{
		PetId:   m.Id(),
		OwnerId: m.OwnerId(),
		Type:    pet.StatusEventTypeRevived,
		Body: pet.RevivedStatusEventBody{
			Slot:          m.Slot(),
			Expiration:    m.Expiration(),
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func reviveFailedEventProvider(petId uint32, ownerId uint32, reason string, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	// mirror evolvedEventProvider's key construction
	value := &pet.StatusEvent[pet.ReviveFailedStatusEventBody]{
		PetId:   petId,
		OwnerId: ownerId,
		Type:    pet.StatusEventTypeReviveFailed,
		Body: pet.ReviveFailedStatusEventBody{
			Reason:        reason,
			TransactionId: transactionId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 6: Implement `Revive` / `ReviveAndEmit`**

In `services/atlas-pets/atlas.com/pets/pet/processor.go`, modelled on `Evolve` / `EvolveAndEmit`:

```go
func (p *ProcessorImpl) ReviveAndEmit(transactionId uuid.UUID, actorId uint32, petId uint32, sourceTemplateId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			return p.With(WithTransaction(tx)).Revive(mb)(transactionId, actorId, petId, sourceTemplateId)
		})
	})
}

// Revive restores a dried-up pet's lifespan from the consumed Water of Life's
// own WZ info/life. It is a SET, not an add: the old expiration is in the past
// by definition, so adding to it would be wrong.
//
// The RESET_PET_EXPIRATION cascade is buffered INSIDE this transaction's outbox
// rather than being a separate saga step. Two mutation steps can half-apply,
// and a half-application is exactly the bug FR-5.4 names: a pet alive here and
// still a doll in the item slot. Buffering the cascade in the same transaction
// makes the pair atomic at the database level; the saga is left responsible
// only for the cross-service consume/refund pair (design §7.1).
//
// Every rejection buffers REVIVE_FAILED and returns nil, NOT an error — the
// transactional emit path discards the buffer when the closure errors, so a
// rejection returned as an error would never reach the saga and the player's
// already-consumed Water of Life would wait out the saga timeout for its refund.
func (p *ProcessorImpl) Revive(mb *message.Buffer) func(transactionId uuid.UUID, actorId uint32, petId uint32, sourceTemplateId uint32) error {
	return func(transactionId uuid.UUID, actorId uint32, petId uint32, sourceTemplateId uint32) error {
		p.l.Debugf("Reviving pet [%d] for character [%d] with source [%d].", petId, actorId, sourceTemplateId)

		pe, err := p.GetById(petId)
		if err != nil {
			p.l.WithError(err).Warnf("Unable to resolve pet [%d] for revive.", petId)
			return mb.Put(pet.EnvStatusEventTopic, reviveFailedEventProvider(petId, actorId, "pet not found", transactionId))
		}
		if pe.OwnerId() != actorId {
			p.l.Warnf("Character [%d] attempted to revive pet [%d] owned by [%d].", actorId, petId, pe.OwnerId())
			return mb.Put(pet.EnvStatusEventTopic, reviveFailedEventProvider(petId, actorId, "pet not owned by character", transactionId))
		}

		// Idempotency / liveness gate (design §9). Redelivery re-cascades the
		// STORED expiration rather than a recomputed one, so the pair converges
		// even if the first delivery's cascade was itself lost.
		if rt := pe.ReviveTransactionId(); rt != nil && *rt == transactionId {
			p.l.Infof("Revive of pet [%d] for transaction [%s] is a redelivery; re-emitting without a write.", petId, transactionId)
			if err = p.ip.ResetPetExpiration(mb)(transactionId, pe.OwnerId(), petId, pe.Expiration(), sourceTemplateId); err != nil {
				return err
			}
			return mb.Put(pet.EnvStatusEventTopic, revivedEventProvider(pe, transactionId))
		}
		if !pe.Expiration().IsZero() && time.Now().Before(pe.Expiration()) {
			p.l.Warnf("Character [%d] attempted to revive pet [%d], which has not dried up.", actorId, petId)
			return mb.Put(pet.EnvStatusEventTopic, reviveFailedEventProvider(petId, pe.OwnerId(), "pet has not dried up", transactionId))
		}

		cd, err := p.cdp.GetById(sourceTemplateId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to resolve cash data for source [%d]; refusing to revive pet [%d].", sourceTemplateId, petId)
			return mb.Put(pet.EnvStatusEventTopic, reviveFailedEventProvider(petId, pe.OwnerId(), "unable to resolve source item data", transactionId))
		}
		if cd.Life() == 0 {
			p.l.Errorf("Source item [%d] has no info/life; refusing to revive pet [%d].", sourceTemplateId, petId)
			return mb.Put(pet.EnvStatusEventTopic, reviveFailedEventProvider(petId, pe.OwnerId(), "source item grants no lifespan", transactionId))
		}

		expiration := time.Now().Add(time.Duration(cd.Life()) * 24 * time.Hour)
		updated, err := Clone(pe).
			SetExpiration(expiration).
			SetReviveTransactionId(&transactionId).
			Build()
		if err != nil {
			return err
		}
		if err = updateOnRevive(p.db)(petId, expiration, transactionId); err != nil {
			return err
		}
		// The cascade: atlas-inventory holds the expiration the CLIENT reads.
		// CUIToolTip::GetPetDeadDate (gms_v83 @0x8ebfde) formats "dried up" or a
		// date from the ITEM SLOT alone, so a pet revived only here would still
		// read as a doll.
		if err = p.ip.ResetPetExpiration(mb)(transactionId, pe.OwnerId(), petId, expiration, sourceTemplateId); err != nil {
			return err
		}
		p.l.Infof("Revived pet [%d] for character [%d]: source [%d], life [%d] days, expiration [%s].", petId, actorId, sourceTemplateId, cd.Life(), expiration)
		return mb.Put(pet.EnvStatusEventTopic, revivedEventProvider(updated, transactionId))
	}
}
```

`p.cdp` is the new cash-data processor field on `ProcessorImpl`. Add it to the struct and initialise it in `NewProcessor` alongside the existing `p.dp` (pet-data) and `p.ip` (inventory) fields, following whatever `With…` option idiom the struct already uses so tests can inject the mock from Task 8. Read `NewProcessor` and the existing `With*` options first, and add a `WithCashDataProcessor(cash.Processor)` option if — and only if — that is how `dp`/`ip` are already overridden in tests; otherwise match the existing mechanism exactly rather than introducing a second one.

Note `updateOnRevive(p.db)`: `Revive` runs under `p.With(WithTransaction(tx))`, so `p.db` is already the transaction handle, matching how `Evolve` calls `updateOnEvolve(tx)` from inside its own `ExecuteTransaction`. Verify which handle is in scope at the call site and use it — a second pooled connection taken from inside a transaction deadlocks at pool size 1.

- [ ] **Step 7: Add the consumer arm**

In `services/atlas-pets/atlas.com/pets/kafka/consumer/pet/consumer.go`, register alongside `handleEvolveCommand` in `InitHandlers`:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleReviveCommand(db)))); err != nil {
				return err
			}
```

and add the handler beside `handleEvolveCommand`:

```go
func handleReviveCommand(db *gorm.DB) message.Handler[pet2.Command[pet2.ReviveCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c pet2.Command[pet2.ReviveCommandBody]) {
		if c.Type != pet2.CommandPetRevive {
			return
		}
		err := pet.NewProcessor(l, ctx, db).ReviveAndEmit(c.TransactionId, c.ActorId, c.PetId, c.Body.SourceTemplateId)
		if err != nil {
			l.WithError(err).Errorf("Unable to revive pet [%d].", c.PetId)
		}
	}
}
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd services/atlas-pets/atlas.com/pets && go test -race ./... `
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/
git commit -m "feat(task-228): implement the atlas-pets REVIVE command with the inventory cascade and idempotency gate"
```

---

### Task 10: `atlas-inventory` — `RESET_PET_EXPIRATION` (design §8.2)

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/data/cash/{model,builder,rest}.go` (add `Life`)
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go`
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`
- Modify: `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go`
- Test: `services/atlas-inventory/atlas.com/inventory/compartment/processor_reset_pet_expiration_test.go` (create)
- Test: `services/atlas-inventory/atlas.com/inventory/data/cash/processor_test.go` (extend)

**Interfaces:**
- Consumes: Task 3's `life` REST field; Task 8's `CommandResetPetExpiration` / `ResetPetExpirationCommandBody` (this task holds the **mirror** — field names and json tags must match Task 8's copy byte-for-byte).
- Produces: `compartment.ProcessorImpl.ResetPetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error` and `ResetPetExpirationAndEmit(...)` with the same trailing parameters. Not consumed by later tasks (reached over Kafka).

- [ ] **Step 1: Extend the cash-data model with `Life`**

`services/atlas-inventory/atlas.com/inventory/data/cash/model.go`:

```go
type Model struct {
	id      uint32
	addTime uint32
	maxDays uint32
	life    uint32
}
```

```go
// Life is info/life in DAYS — the lifespan a pet-revival item (Water of Life,
// classification 518) grants. 0518.img carries life but NO maxDays, which is
// exactly why EXTEND_EXPIRATION's maxDays ceiling cannot serve the revive flow.
func (m Model) Life() uint32 { return m.life }
```

`builder.go`: add the `life` field, `func (b *ModelBuilder) SetLife(v uint32) *ModelBuilder { b.life = v; return b }`, and `life: b.life` in `Build`.

`rest.go`: add `Life uint32 \`json:"life,omitempty"\`` to `RestModel` and `life: rm.Life` to `Extract`.

- [ ] **Step 2: Write the failing tests**

Extend `services/atlas-inventory/atlas.com/inventory/data/cash/processor_test.go` with a case asserting a `{"life":90}` attributes payload yields `m.Life() == 90`.

Create `services/atlas-inventory/atlas.com/inventory/compartment/processor_reset_pet_expiration_test.go`, reusing the setup style of the existing `ExtendAssetExpiration` tests in this package. Assert:

1. **Resolves by petId.** Given a cash compartment holding two pet assets (petIds 7 and 9) plus a non-pet cash asset, `ResetPetExpiration` for petId 9 updates only asset 9's expiration and emits exactly one asset `UPDATED`.
2. **`life == 0` rejects without mutating.** Cash stub returning no `life`: the call returns an error, no asset row changed, no `UPDATED` emitted.
3. **Beyond the re-derived cap rejects without mutating.** Cash stub returning `life: 90`, requested expiration `now + 200 days`: error returned, asset unchanged. This is the NFR-2 forged-expiration guard.
4. **Within the cap succeeds.** `life: 90`, requested `now + 90 days` less a minute: applied, `UPDATED` emitted.
5. **Idempotent redelivery.** Calling twice with the identical absolute expiration leaves the asset unchanged on the second call but still emits `UPDATED`, so the pair converges instead of stalling. (This falls out of `asset.ExtendExpiration`'s equal-value branch — the test pins it.)
6. **Unknown petId errors** and mutates nothing.

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test ./compartment/ -run ResetPetExpiration -v`
Expected: FAIL — `undefined: ResetPetExpiration`

- [ ] **Step 4: Add the command contract (mirror)**

`services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go` — add to the const block after `CommandExtendExpiration`:

```go
	CommandResetPetExpiration = "RESET_PET_EXPIRATION"
```

and the body after `ExtendExpirationCommandBody`:

```go
// ResetPetExpirationCommandBody sets a dried-up pet asset's expiration to an
// absolute instant. The asset is resolved by (CharacterId, PetId) — never by
// slot — mirroring ChangeTemplateCommandBody. SourceTemplateId names the
// consumed Water of Life so this service can re-derive the ceiling itself; the
// caller is not a trust boundary. Absolute (not a duration) so a redelivered
// command is a no-op rather than a second grant.
//
// MIRROR: this struct is duplicated in
// services/atlas-pets/.../kafka/message/compartment/kafka.go (the producer).
// The two live in separate Go modules, so a field name or json tag changed in
// one and not the other fails no build — it decodes into a zero-valued body at
// runtime: a pet revived to the zero time, i.e. still a doll.
type ResetPetExpirationCommandBody struct {
	PetId            uint32    `json:"petId"`
	Expiration       time.Time `json:"expiration"`
	SourceTemplateId uint32    `json:"sourceTemplateId"`
}
```

- [ ] **Step 5: Implement the processor**

In `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`, beside `ChangeTemplate` (whose structure it mirrors) and using `ExtendAssetExpiration`'s cap idiom:

```go
func (p *ProcessorImpl) ResetPetExpirationAndEmit(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			return p.WithTransaction(tx).ResetPetExpiration(mb)(transactionId, characterId, petId, expiration, sourceTemplateId)
		})
	})
}

// ResetPetExpiration sets a dried-up pet asset's expiration to an absolute
// instant, rejecting the request outright if it exceeds a cap this service
// re-derives itself.
//
// atlas-pets computes the expiration, but it is NOT a trust boundary: a forged
// COMMAND_TOPIC_COMPARTMENT message could otherwise set an arbitrary
// expiration. The cap is re-derived here from the consumed Water of Life's own
// cash data (info/life, in days), anchored to now. A request beyond that cap is
// REJECTED, not clamped — the same reasoning as ExtendAssetExpiration: by the
// time this runs the water has already been consumed by the saga's first step,
// and rejecting produces a full refund via the compensator rather than a
// silent, unauditable partial grant.
//
// EXTEND_EXPIRATION is deliberately NOT reused: it hard-rejects maxDays == 0,
// and 0518.img has no maxDays node at all. Relaxing that guard to accept a
// second cap source would make one command mean two things and weaken the
// extender's own ceiling.
func (p *ProcessorImpl) ResetPetExpiration(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, expiration time.Time, sourceTemplateId uint32) error {
		p.l.Debugf("Character [%d] resetting expiration of pet [%d] asset with source [%d].", characterId, petId, sourceTemplateId)

		cd, err := p.cashProcessor.GetById(sourceTemplateId)
		if err != nil {
			p.l.WithError(err).Errorf("Character [%d] unable to resolve source [%d] cash data; refusing to reset pet expiration.", characterId, sourceTemplateId)
			return err
		}
		if cd.Life() == 0 {
			p.l.Errorf("Character [%d] source [%d] has no info/life; refusing to reset pet expiration.", characterId, sourceTemplateId)
			return errors.New("source item grants no lifespan")
		}
		serverCap := time.Now().Add(time.Duration(cd.Life()) * 24 * time.Hour)
		if expiration.After(serverCap) {
			p.l.Warnf("Character [%d] requested pet expiration [%s] beyond the server-derived cap [%s] for source [%d]; rejecting.", characterId, expiration, serverCap, sourceTemplateId)
			return errors.New("requested expiration exceeds the source item's server-derived cap")
		}

		invLock := LockRegistry().Get(characterId, inventory.TypeValueCash)
		invLock.Lock()
		defer invLock.Unlock()

		return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			cp := p.WithTransaction(tx).WithAssetProcessor(asset.NewProcessor(p.l, p.ctx, tx))
			c, err := cp.GetByCharacterAndType(characterId)(inventory.TypeValueCash)
			if err != nil {
				return err
			}
			c, err = cp.DecorateAsset(c)
			if err != nil {
				return err
			}
			for _, a := range c.Assets() {
				if a.IsPet() && a.PetId() == petId {
					return cp.assetProcessor.ExtendExpiration(mb)(transactionId, characterId)(a, expiration)
				}
			}
			return fmt.Errorf("pet [%d] asset not found in cash compartment for character [%d]", petId, characterId)
		})
	}
}
```

`asset.ExtendExpiration` is reused as-is: its four guards are exactly right here — it refuses a locked asset, refuses a permanent (zero-expiration) asset, refuses a backwards move, and **no-ops with an `UPDATED` emit when the value is already equal**, which is what makes redelivery terminate cleanly instead of stalling the saga.

- [ ] **Step 6: Add the consumer arm**

In `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go`, register alongside `handleExtendExpirationCommand`:

```go
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleResetPetExpirationCommand(db)))); err != nil {
				return err
			}
```

and add:

```go
func handleResetPetExpirationCommand(db *gorm.DB) message.Handler[compartment2.Command[compartment2.ResetPetExpirationCommandBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, c compartment2.Command[compartment2.ResetPetExpirationCommandBody]) {
		if c.Type != compartment2.CommandResetPetExpiration {
			return
		}

		l.Debugf("Received RESET_PET_EXPIRATION command for character [%d], pet [%d], source [%d].",
			c.CharacterId, c.Body.PetId, c.Body.SourceTemplateId)

		err := compartment.NewProcessor(l, ctx, db).ResetPetExpirationAndEmit(
			c.TransactionId,
			c.CharacterId,
			c.Body.PetId,
			c.Body.Expiration,
			c.Body.SourceTemplateId,
		)
		if err != nil {
			l.WithError(err).Errorf("Unable to reset expiration of pet [%d] asset for character [%d].", c.Body.PetId, c.CharacterId)
		}
	}
}
```

- [ ] **Step 7: Verify the two mirror copies are identical**

Run:

```bash
diff <(sed -n '/^type ResetPetExpirationCommandBody/,/^}/p' services/atlas-pets/atlas.com/pets/kafka/message/compartment/kafka.go) \
     <(sed -n '/^type ResetPetExpirationCommandBody/,/^}/p' services/atlas-inventory/atlas.com/inventory/kafka/message/compartment/kafka.go)
```

Expected: no output (identical field names, types and json tags).

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd services/atlas-inventory/atlas.com/inventory && go test -race ./... `
Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/
git commit -m "feat(task-228): add RESET_PET_EXPIRATION with an independently re-derived lifespan cap"
```

---

### Task 11: `atlas-saga-orchestrator` — the `revive_pet` step (design §7.2)

**Files:**
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/model.go` (Type + Action + Payload alias blocks; the payload `switch` ~line 1404)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance.go` (EventKind consts ~line 54; `acceptanceTable` ~line 154)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go` (`allActions`)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/handler.go` (interface ~line 117; dispatch `switch` ~line 828; new `handleRevivePet` beside `handleEvolvePet` ~line 1401)
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/processor.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/pet/producer.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/pet/kafka.go`
- Modify: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/consumer/pet/consumer.go`

**Interfaces:**
- Consumes: Task 6's `sharedsaga.RevivePet` / `RevivePetPayload` / `PetRevive`; Task 8's `REVIVE` command shape and `REVIVED` / `REVIVE_FAILED` status events (mirrored in this module's own `kafka/message/pet/kafka.go`).
- Produces: acceptance of a `revive_pet` step by either event. Not consumed by later tasks.

- [ ] **Step 1: Write the failing tests**

Add `sharedsaga.RevivePet` to the `allActions` slice in `event_acceptance_test.go`, on the line directly below `sharedsaga.EvolvePet,`:

```go
	sharedsaga.EvolvePet, sharedsaga.RevivePet,
```

and add the expectation row to the table in the same file beside `{sharedsaga.EvolvePet, EventKindPetEvolved}`:

```go
		{sharedsaga.RevivePet, EventKindPetRevived},
		{sharedsaga.RevivePet, EventKindPetReviveFailed},
```

Read that test's table shape first — if it asserts a single kind per action, add only the row that fits its signature and cover the second kind with a direct `acceptanceTable[sharedsaga.RevivePet]` membership assertion instead.

Add to `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/accept_event_test.go`, modelled on `TestAcceptEvent_EvolvePetMatch` (line 133):

```go
func newRevivePetSaga(t *testing.T, ctx context.Context, tx uuid.UUID) {
	t.Helper()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(PetRevive).
		SetInitiatedBy("test").
		AddStep("revive", Pending, RevivePet, RevivePetPayload{
			CharacterId:      1,
			PetId:            2,
			SourceTemplateId: 5180000,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)
}

func TestAcceptEvent_RevivePetMatchesRevived(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	newRevivePetSaga(t, ctx, tx)

	decision, ok := p.AcceptEvent(tx, EventKindPetRevived)
	require.True(t, ok, "REVIVED event must complete a pending revive_pet step")
	assert.Equal(t, "revive", decision.Step.StepId())
	assert.Equal(t, RevivePet, decision.Step.Action())
	assert.Equal(t, tx, decision.Saga.TransactionId())
}

func TestAcceptEvent_RevivePetMatchesReviveFailed(t *testing.T) {
	// REVIVE_FAILED must be accepted too, so the saga compensates the
	// already-completed destroy step immediately rather than waiting out the
	// flat timeout with the player's Water of Life gone.
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	newRevivePetSaga(t, ctx, tx)

	decision, ok := p.AcceptEvent(tx, EventKindPetReviveFailed)
	require.True(t, ok, "REVIVE_FAILED event must be accepted by a pending revive_pet step")
	assert.Equal(t, "revive", decision.Step.StepId())
	assert.Equal(t, RevivePet, decision.Step.Action())
}

func TestAcceptEvent_RevivePetRejectsClosenessChanged(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	newRevivePetSaga(t, ctx, tx)

	_, ok := p.AcceptEvent(tx, EventKindPetClosenessChanged)
	assert.False(t, ok, "CLOSENESS_CHANGED must not complete a revive_pet step")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test ./saga/ -run 'RevivePet|AcceptanceTable' -v`
Expected: FAIL — `undefined: EventKindPetRevived`

- [ ] **Step 3: Add the aliases and event kinds**

`saga/model.go` — add to the saga-`Type` alias block: `PetRevive = sharedsaga.PetRevive`. Add to the `Action` alias block (beside `EvolvePet = sharedsaga.EvolvePet`): `RevivePet = sharedsaga.RevivePet`. Add to the payload alias block (beside `EvolvePetPayload`): `RevivePetPayload = sharedsaga.RevivePetPayload`.

`saga/model.go` — add the payload arm directly after `case EvolvePet:` in the payload `switch` (~line 1404):

```go
	case RevivePet:
		var payload RevivePetPayload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return err
		}
		st.payload = payload
```

Copy the exact body of the neighbouring `case EvolvePet:` arm — variable names and error handling in that switch differ from `libs/atlas-saga/unmarshal.go`; match the local one.

`saga/event_acceptance.go` — add beside `EventKindPetEvolved`:

```go
	EventKindPetRevived      EventKind = "pet.revived"
	EventKindPetReviveFailed EventKind = "pet.revive_failed"
```

and to `acceptanceTable` directly after the `sharedsaga.EvolvePet` row:

```go
	sharedsaga.RevivePet:              {EventKindPetRevived, EventKindPetReviveFailed},
```

- [ ] **Step 4: Add the command producer and processor method**

`kafka/message/pet/kafka.go` (this module's mirror) — add `CommandPetRevive = "REVIVE"`, `ReviveCommandBody{SourceTemplateId uint32 \`json:"sourceTemplateId"\`}`, `StatusEventTypeRevived = "REVIVED"`, `StatusEventTypeReviveFailed = "REVIVE_FAILED"`, `RevivedStatusEventBody` and `ReviveFailedStatusEventBody` with field names and json tags matching Task 8's copies byte-for-byte. Check what this file already names its topic constant (`EnvEventTopicPetStatus` here vs `EnvStatusEventTopic` in atlas-pets) and keep the local name.

`pet/producer.go` — after `EvolveProvider`:

```go
func ReviveProvider(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(petId))
	value := &pet2.Command[pet2.ReviveCommandBody]{
		TransactionId: transactionId,
		ActorId:       characterId,
		PetId:         petId,
		Type:          pet2.CommandPetRevive,
		Body: pet2.ReviveCommandBody{
			SourceTemplateId: sourceTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

`ActorId` is load-bearing: atlas-pets rejects a revive whose `ActorId` does not own the pet.

`pet/processor.go` — add to the `Processor` interface and implement:

```go
	ReviveAndEmit(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error
	Revive(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error
```

```go
func (p *ProcessorImpl) ReviveAndEmit(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Revive(mb)(transactionId, characterId, petId, sourceTemplateId)
	})
}

func (p *ProcessorImpl) Revive(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error {
	return func(transactionId uuid.UUID, characterId uint32, petId uint32, sourceTemplateId uint32) error {
		return mb.Put(pet2.EnvCommandTopic, ReviveProvider(transactionId, characterId, petId, sourceTemplateId))
	}
}
```

- [ ] **Step 5: Add the step handler**

`saga/handler.go` — add `handleRevivePet(s Saga, st Step[any]) error` to the `Handler` interface beside `handleEvolvePet`; add the dispatch arm after `case EvolvePet:`:

```go
	case RevivePet:
		return h.handleRevivePet, true
```

and the implementation after `handleEvolvePet`:

```go
func (h *HandlerImpl) handleRevivePet(s Saga, st Step[any]) error {
	payload, ok := st.Payload().(RevivePetPayload)
	if !ok {
		return errors.New("invalid payload")
	}

	err := h.petP.ReviveAndEmit(s.TransactionId(), payload.CharacterId, payload.PetId, payload.SourceTemplateId)
	if err != nil {
		h.logActionError(s, st, err, "Unable to revive pet.")
		return err
	}

	return nil
}
```

If `Handler` has a generated or hand-written mock in this package, add the method there too or the package will not compile.

- [ ] **Step 6: Add the two consumer arms**

`kafka/consumer/pet/consumer.go` — register two more handlers in `InitHandlers` alongside `handleEvolvedEvent`, and add, modelled on `handleEvolvedEvent` and on `note`'s `handleCreateFailedEvent`:

```go
func handleRevivedEvent(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.RevivedStatusEventBody]) {
	if e.Type != pet2.StatusEventTypeRevived {
		return
	}
	if e.Body.TransactionId == uuid.Nil {
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.Body.TransactionId, saga.EventKindPetRevived); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"pet_id":         e.PetId,
		"owner_id":       e.OwnerId,
		"expiration":     e.Body.Expiration,
	}).Debug("Pet revived successfully, marking saga step as completed")

	_ = p.StepCompleted(e.Body.TransactionId, true)
}

func handleReviveFailedEvent(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.ReviveFailedStatusEventBody]) {
	if e.Type != pet2.StatusEventTypeReviveFailed {
		return
	}
	if e.Body.TransactionId == uuid.Nil {
		return
	}

	p := saga.NewProcessor(l, ctx)
	if _, ok := p.AcceptEvent(e.Body.TransactionId, saga.EventKindPetReviveFailed); !ok {
		return
	}

	l.WithFields(logrus.Fields{
		"transaction_id": e.Body.TransactionId.String(),
		"pet_id":         e.PetId,
		"owner_id":       e.OwnerId,
		"reason":         e.Body.Reason,
	}).Warn("Pet revive failed, failing saga step so the Water of Life is refunded.")

	_ = p.StepCompleted(e.Body.TransactionId, false)
}
```

`StepCompleted(…, false)` is what drives the reverse-walk compensation of the already-completed `destroy_water_of_life` step — the refund-by-template-id path `DestroyAsset` compensation already provides.

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-saga-orchestrator/atlas.com/saga-orchestrator && go test -race ./... `
Expected: all PASS

- [ ] **Step 8: Commit**

```bash
git add services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/
git commit -m "feat(task-228): add the revive_pet saga step with REVIVED/REVIVE_FAILED acceptance"
```

---

### Task 12: `atlas-channel` — the `WaterOfLifeHandle` handler (FR-3.1, FR-4, FR-6)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/water_of_life.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/water_of_life_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (handler map, beside line 923's `ChalkboardCloseHandle` registration)
- Modify: `services/atlas-channel/atlas.com/channel/data/cash/rest.go` (add `Life`)
- Modify: `services/atlas-channel/atlas.com/channel/saga/model.go` (add the `PetRevive` and `RevivePet` aliases)

**Interfaces:**
- Consumes: Task 1's `item.ClassificationWaterOfLife`; Task 4's `serverbound.WaterOfLife` / `WaterOfLifeHandle`; Task 6's `saga.PetRevive` / `saga.RevivePet` / `saga.RevivePetPayload`; Task 3's `life` REST field; existing `pet.Processor.GetByOwner`, `character.Processor.GetById(cp.InventoryDecorator)`, `saga.Processor.Create`.
- Produces: `handler.WaterOfLifeHandleFunc`. Consumed by `main.go` in this task.

- [ ] **Step 1: Write the failing pure-logic tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/water_of_life_test.go`. Both units under test are pure, so no session or REST plumbing is needed. Build `pet.Model` values with `pet.NewModelBuilder(...)` / the package's Builder — never a struct literal.

```go
package handler

import (
	"atlas-channel/pet"
	"testing"
	"time"
)

func TestSelectRevivableTargetPicksLatestPastExpiration(t *testing.T) {
	now := time.Now()
	// Two dolls; the one that dried up MOST RECENTLY wins (FR-4.2).
	older := buildPet(t, 1, now.Add(-72*time.Hour))
	newer := buildPet(t, 2, now.Add(-1*time.Hour))
	live := buildPet(t, 3, now.Add(24*time.Hour))

	got, ok := selectRevivableTarget([]pet.Model{older, live, newer}, now)
	if !ok {
		t.Fatal("expected a target")
	}
	if got.Id() != 2 {
		t.Fatalf("selected pet %d, want 2 (latest past expiration)", got.Id())
	}
}

func TestSelectRevivableTargetTieBreaksOnLowestId(t *testing.T) {
	now := time.Now()
	exp := now.Add(-1 * time.Hour)
	// Two pets bought in one transaction share an expiration -- the norm, not
	// an edge case. The choice must be reproducible.
	got, ok := selectRevivableTarget([]pet.Model{buildPet(t, 9, exp), buildPet(t, 4, exp)}, now)
	if !ok {
		t.Fatal("expected a target")
	}
	if got.Id() != 4 {
		t.Fatalf("selected pet %d, want 4 (lowest id on a tie)", got.Id())
	}
}

func TestSelectRevivableTargetIgnoresLiveAndPermanentPets(t *testing.T) {
	now := time.Now()
	live := buildPet(t, 1, now.Add(24*time.Hour))
	permanent := buildPet(t, 2, time.Time{})

	if _, ok := selectRevivableTarget([]pet.Model{live, permanent}, now); ok {
		t.Fatal("expected no target when no pet has dried up")
	}
}

func TestSelectRevivableTargetEmpty(t *testing.T) {
	if _, ok := selectRevivableTarget(nil, time.Now()); ok {
		t.Fatal("expected no target for a character with no pets")
	}
}
```

Write `buildPet(t *testing.T, id uint32, expiration time.Time) pet.Model` in the test file using `pet.NewModelBuilder`, reading `services/atlas-channel/atlas.com/channel/pet/builder.go` for its exact constructor signature.

Add tests for the item-location half. `asset.NewBuilder(compartmentId uuid.UUID, templateId uint32)` with `.SetId`, `.SetSlot` and `.Build()` is the constructor (`services/atlas-channel/atlas.com/channel/asset/builder.go:119-139`):

```go
func buildCashAsset(id uint32, templateId uint32, slot int16) asset.Model {
	return asset.NewBuilder(uuid.New(), templateId).SetId(id).SetSlot(slot).Build()
}

func TestFindWaterOfLifePicksLowestSlot(t *testing.T) {
	// Two classification-518 assets: the lower slot wins, so the choice is
	// reproducible regardless of the backing slice's order.
	assets := []asset.Model{
		buildCashAsset(2, 5180000, 9),
		buildCashAsset(1, 5180000, 3),
	}
	got, ok := findWaterOfLife(assets)
	if !ok {
		t.Fatal("expected to find a Water of Life")
	}
	if got != 5180000 {
		t.Fatalf("templateId = %d, want 5180000", got)
	}
}

func TestFindWaterOfLifeIgnoresOtherCashClassifications(t *testing.T) {
	// 517 is a pet name tag and 519 a pet skill item; neither is a Water of Life.
	assets := []asset.Model{
		buildCashAsset(1, 5170000, 1),
		buildCashAsset(2, 5190000, 2),
		buildCashAsset(3, 5000000, 3),
	}
	if _, ok := findWaterOfLife(assets); ok {
		t.Fatal("expected no Water of Life among neighbouring cash classifications")
	}
}

func TestFindWaterOfLifeNoneHeld(t *testing.T) {
	if _, ok := findWaterOfLife(nil); ok {
		t.Fatal("expected no Water of Life in an empty compartment")
	}
}
```

If `Build()` on that builder returns `(Model, error)` rather than a bare `Model`, adjust `buildCashAsset` to take `t *testing.T` and fail on the error — read the builder's `Build` signature before writing the helper.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'WaterOfLife|RevivableTarget' -v`
Expected: FAIL — `undefined: selectRevivableTarget`

- [ ] **Step 3: Add the `Life` field to the channel's cash client**

`services/atlas-channel/atlas.com/channel/data/cash/rest.go` — add to `RestModel`:

```go
	// Life is info/life in DAYS served by atlas-data: the lifespan a Water of
	// Life (classification 518) grants a revived pet. 0 or absent means the WZ
	// node is missing, which the handler treats as "reject, consume nothing".
	// The channel reads it ONLY as a pre-flight check -- the authoritative
	// derivation happens in atlas-pets and is re-bounded in atlas-inventory.
	Life uint32 `json:"life,omitempty"`
```

- [ ] **Step 4: Add the saga aliases**

`services/atlas-channel/atlas.com/channel/saga/model.go` — add `PetRevive = sharedsaga.PetRevive` to the saga-type const block, and `RevivePet = sharedsaga.RevivePet` plus `RevivePetPayload = sharedsaga.RevivePetPayload` to the action and payload alias blocks (find those blocks by grepping for `EvolvePet` / `ExtendAssetExpirationPayload` in that file and add beside them).

- [ ] **Step 5: Write the handler**

Create `services/atlas-channel/atlas.com/channel/socket/handler/water_of_life.go`:

```go
package handler

import (
	"atlas-channel/character"
	"atlas-channel/data/cash"
	"atlas-channel/pet"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	petsb "github.com/Chronicle20/atlas/libs/atlas-packet/pet/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// Rejection texts. All three live here so the synchronous path below and the
// asynchronous REVIVE_FAILED path (kafka/consumer/pet) cannot drift.
const (
	waterOfLifeNoItemMessage   = "You do not have a Water of Life."
	waterOfLifeNoDollMessage   = "You have no pet that has dried up."
	waterOfLifeNoEffectMessage = "The Water of Life has no effect."
	// WaterOfLifeFailedMessage is announced when atlas-pets rejects the revive
	// after the item was already consumed; the saga refunds it.
	WaterOfLifeFailedMessage = "The Water of Life had no effect. It has been returned to you."
)

// WaterOfLifeHandleFunc handles CWvsContext::SendWaterOfLife.
//
// The packet body is EMPTY, so the server derives every operand itself: the
// held Water of Life and the most-recently-dried-up pet. It is a top-level
// opcode handler, NOT an arm of CharacterCashItemUseHandleFunc -- the client
// reaches SendWaterOfLife from SendEtcCashItemUseRequest (gms_v83 @0xa1dc5b),
// a sibling of SendCashSlotItemUseRequest and SendConsumeCashItemUseRequest,
// so the cash-item-use dispatcher never observes this item.
//
// NO EnableActions on any path. SendConsumeCashItemUseRequest (@0xa0ea6f) is
// the ONLY caller of CWvsContext::SetExclRequestSent (@0xa0ebbc); the CASH-tab
// double-click path gates on CanSendExclRequest(500) (CDraggableItem::
// OnDoubleClicked @0x4efdf7) but never latches. Sending an unlock here would be
// inert, and a lie in the code about what the client is doing.
func WaterOfLifeHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := petsb.WaterOfLife{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		reject := func(text string) {
			_ = session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageOperationSystemMessageBody(text))(s)
		}

		cp := character.NewProcessor(l, ctx)
		c, err := cp.GetById(cp.InventoryDecorator)(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] unable to resolve inventory for a Water of Life use.", s.CharacterId())
			reject(waterOfLifeNoEffectMessage)
			return
		}
		sourceTemplateId, ok := findWaterOfLife(c.Inventory().Cash().Assets())
		if !ok {
			l.Warnf("Character [%d] used a Water of Life while holding none.", s.CharacterId())
			reject(waterOfLifeNoItemMessage)
			return
		}

		ps, err := pet.NewProcessor(l, ctx).GetByOwner(s.CharacterId())
		if err != nil {
			l.WithError(err).Warnf("Character [%d] unable to resolve pets for a Water of Life use.", s.CharacterId())
			reject(waterOfLifeNoEffectMessage)
			return
		}
		target, ok := selectRevivableTarget(ps, time.Now())
		if !ok {
			l.Warnf("Character [%d] used a Water of Life with no dried-up pet.", s.CharacterId())
			reject(waterOfLifeNoDollMessage)
			return
		}

		// Pre-flight only (FR-8.3): the authoritative derivation happens in
		// atlas-pets and is independently re-bounded in atlas-inventory. This
		// read exists so a WZ data error costs the player nothing -- once the
		// saga starts, the item is gone.
		cd, err := cash.NewProcessor(l, ctx).GetById(sourceTemplateId)
		if err != nil {
			l.WithError(err).Warnf("Character [%d] unable to resolve cash data for Water of Life [%d].", s.CharacterId(), sourceTemplateId)
			reject(waterOfLifeNoEffectMessage)
			return
		}
		if cd.Life == 0 {
			l.Errorf("Water of Life [%d] has no info/life; refusing to consume it for character [%d].", sourceTemplateId, s.CharacterId())
			reject(waterOfLifeNoEffectMessage)
			return
		}

		l.Infof("Character [%d] reviving pet [%d] with Water of Life [%d] (life [%d] days).", s.CharacterId(), target.Id(), sourceTemplateId, cd.Life)

		transactionId := uuid.New()
		now := time.Now()
		_ = saga.NewProcessor(l, ctx).Create(saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.PetRevive,
			InitiatedBy:   "WATER_OF_LIFE",
			Steps: []saga.Step{
				{
					StepId: "destroy_water_of_life",
					Status: saga.Pending,
					Action: saga.DestroyAsset,
					Payload: saga.DestroyAssetPayload{
						CharacterId: s.CharacterId(),
						TemplateId:  sourceTemplateId,
						Quantity:    1,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "revive_pet",
					Status: saga.Pending,
					Action: saga.RevivePet,
					Payload: saga.RevivePetPayload{
						CharacterId:      s.CharacterId(),
						PetId:            target.Id(),
						SourceTemplateId: sourceTemplateId,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		})
	}
}

// findWaterOfLife returns the template id of the character's Water of Life,
// resolved by CLASSIFICATION (518) so every present and future template
// qualifies. The lowest slot wins, so the choice is reproducible regardless of
// the backing slice's order. Only existence matters downstream:
// DestroyAssetPayload takes {CharacterId, TemplateId, Quantity} and resolves
// the slot itself.
func findWaterOfLife(assets []asset.Model) (uint32, bool) {
	var (
		best     uint32
		bestSlot int16
		found    bool
	)
	for _, a := range assets {
		if item.GetClassification(item.Id(a.TemplateId())) != item.ClassificationWaterOfLife {
			continue
		}
		if !found || a.Slot() < bestSlot {
			best, bestSlot, found = a.TemplateId(), a.Slot(), true
		}
	}
	return best, found
}

// selectRevivableTarget picks the MOST-RECENTLY-expired pet: among pets whose
// expiration is strictly in the past, the greatest (latest) expiration wins,
// with the lowest pet id breaking ties. The tie-break is not an edge case --
// two pets bought in one transaction share an expiration timestamp, and the
// operation must be reproducible. A zero expiration is a permanent pet, never
// a doll.
func selectRevivableTarget(ps []pet.Model, now time.Time) (pet.Model, bool) {
	candidates := make([]pet.Model, 0, len(ps))
	for _, p := range ps {
		if p.Expiration().IsZero() {
			continue
		}
		if now.After(p.Expiration()) {
			candidates = append(candidates, p)
		}
	}
	if len(candidates) == 0 {
		return pet.Model{}, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].Expiration().Equal(candidates[j].Expiration()) {
			return candidates[i].Expiration().After(candidates[j].Expiration())
		}
		return candidates[i].Id() < candidates[j].Id()
	})
	return candidates[0], true
}
```

Two imports need resolving against the real tree before this compiles: the `charpkt` alias used by `CharacterStatusMessageOperationSystemMessageBody` (copy the import block from `services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer.go`, which makes the identical call at line 208), and the `asset` package `findWaterOfLife` ranges over (whatever `c.Inventory().Cash().Assets()` returns). Read both and fix the import block; do not guess the alias.

- [ ] **Step 6: Register the handler**

`services/atlas-channel/atlas.com/channel/main.go`, in the same block as line 923:

```go
	handlerMap[petsb.WaterOfLifeHandle] = handler.WaterOfLifeHandleFunc
```

Use the file's existing alias for `libs/atlas-packet/pet/serverbound` (grep the import block for other `pet/serverbound` registrations such as `PetSpawnHandle`) rather than adding a second import of the same package.

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./socket/handler/ -run 'WaterOfLife|RevivableTarget' -v`
Expected: build clean, all PASS

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(task-228): add the WaterOfLifeHandle channel handler with server-side target resolution"
```

---

### Task 13: `atlas-channel` announces an asynchronous revive failure (FR-6.3, design §7.3)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/pet/kafka.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go`

**Interfaces:**
- Consumes: Task 8's `REVIVE_FAILED` event shape; Task 12's `handler.WaterOfLifeFailedMessage`.
- Produces: nothing consumed by later tasks.

The saga does the refund; the channel does the talking. That split avoids teaching the saga orchestrator about world/channel routing for one message.

- [ ] **Step 1: Add the event to the channel's mirror**

`services/atlas-channel/atlas.com/channel/kafka/message/pet/kafka.go` — add to the status-event const block:

```go
	StatusEventTypeReviveFailed = "REVIVE_FAILED"
```

and the body, with field names and json tags matching Task 8's copy byte-for-byte:

```go
// ReviveFailedStatusEventBody reports that atlas-pets rejected a Water of Life
// revive after the item was already consumed. The saga refunds the item; this
// channel consumer is what tells the player why nothing happened.
type ReviveFailedStatusEventBody struct {
	Reason        string    `json:"reason"`
	TransactionId uuid.UUID `json:"transactionId"`
}
```

Add the `"github.com/google/uuid"` import if the file lacks it. The channel does not need `REVIVED` — the asset `UPDATED` event already carries the refresh (Task 14's verification covers this).

- [ ] **Step 2: Add the consumer arm**

`services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go` — register alongside the existing arms in `InitHandlers` (append to `handles` in the same `id, err = rf(...)` style), and add the handler modelled on `handleFullnessChanged`'s tenant/server gating:

```go
func handleReviveFailed(sc server.Model, wp writer.Producer) message.Handler[pet2.StatusEvent[pet2.ReviveFailedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e pet2.StatusEvent[pet2.ReviveFailedStatusEventBody]) {
		if e.Type != pet2.StatusEventTypeReviveFailed {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		l.Warnf("Pet [%d] revive failed for character [%d]: %s.", e.PetId, e.OwnerId, e.Body.Reason)

		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.OwnerId,
			session.Announce(l)(ctx)(wp)(charcb.CharacterStatusMessageWriter)(charpkt.CharacterStatusMessageOperationSystemMessageBody(handler.WaterOfLifeFailedMessage)))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce revive failure to character [%d].", e.OwnerId)
		}
	}
}
```

Copy the exact tenant/channel gating and the `charcb` / `charpkt` import aliases from a neighbouring handler in the same file (e.g. `handleFullnessChanged`); the event body carries no world/channel, so `IfPresentByCharacterId` on this channel is the routing — a character on another channel simply hears nothing, which is correct.

If importing `socket/handler` from `kafka/consumer/pet` creates an import cycle, move the four message constants from Task 12's handler file into a small shared package (e.g. `services/atlas-channel/atlas.com/channel/socket/handler/message` or an existing text-constants home) and import that from both sides. Do **not** duplicate the string — the whole point of the shared constant is that the sync and async paths cannot drift.

- [ ] **Step 3: Verify the module builds and tests stay green**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./... `
Expected: build clean, all PASS

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/
git commit -m "feat(task-228): announce an asynchronous pet revive failure to the owner"
```

---

### Task 14: Promote the five `WATER_OF_LIFE` matrix cells (FR-10)

**Files:**
- Modify: `docs/packets/audits/status.json` and `docs/packets/audits/STATUS.md` (regenerated, never hand-edited)
- Modify: `libs/atlas-packet/pet/serverbound/water_of_life_test.go` (evidence-pinned fixtures)
- Create: evidence records under `docs/packets/evidence/`

**Interfaces:**
- Consumes: Task 4's codec and its `packet-audit:verify` markers.
- Produces: five `verified` cells. Not consumed by later tasks.

The current `status.json` row for `WATER_OF_LIFE` has `gms_v83`/`v84`/`v87`/`v92`/`v95` at `state: incomplete` ("no audit report") and `gms_v48`/`v61`/`v72`/`v79`/`jms_v185` at `state: n-a` with `opcode: -1`. The row also has no `packet` field yet — promotion adds `"packet": "pet/serverbound/WaterOfLife"`.

Do **not** restate the verify procedure here. Drive it through the canonical entry point, one cell at a time.

- [ ] **Step 1: Verify each of the five cells**

For each of `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, run the single-cell verify procedure via the `/verify-packet` command (or dispatch the `packet-verifier` agent), which drives [`docs/packets/audits/VERIFYING_A_PACKET.md`](../../packets/audits/VERIFYING_A_PACKET.md):

```
/verify-packet pet/serverbound/WaterOfLife gms_v83
```

…and likewise for `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`.

Per CLAUDE.md's model preference, pin the verifier subagents to Sonnet or Haiku, not an expensive model.

The IDA addresses for each version's sender are already established and are in the codec's `packet-audit:verify` markers: v83 `0xa1dce6`, v84 `0xa68f85`, v87 `0xab501c`, v92 `0x9c6f90`, v95 `0x9f28e0`. Resolve each IDA session from `idb_list` by binary **name**, not by port.

A cell that does not promote is a failure report, not a prose claim — do not mark this step done on the strength of "the codec looks right".

- [ ] **Step 2: Confirm all five promoted and the five `n-a` cells held**

Run:

```bash
python3 -c "
import json
d=json.load(open('docs/packets/audits/status.json'))
r=[x for x in d['rows'] if x.get('op')=='WATER_OF_LIFE'][0]
print('packet:', r.get('packet'))
for k,v in sorted(r['cells'].items()):
    print(k, v['state'], v.get('opcode'))
"
```

Expected: `packet: pet/serverbound/WaterOfLife`; `gms_v83 verified 117`, `gms_v84 verified 117`, `gms_v87 verified 120`, `gms_v92 verified 128`, `gms_v95 verified 129`; `gms_v48 / gms_v61 / gms_v72 / gms_v79 / jms_v185` still `n-a -1`.

- [ ] **Step 3: Run the packet CI gates**

Run the gate set named in [`docs/packets/PROCESS.md`](../../packets/PROCESS.md) — at minimum the matrix regeneration check, the fname-doc check, the `operations` check and the `n-a` consistency gate — and show exit 0 for each. `PROCESS.md` is the source of truth for the current gate list; read it rather than relying on this line.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/ libs/atlas-packet/pet/serverbound/
git commit -m "docs(task-228): promote the five WATER_OF_LIFE matrix cells with fixtures and pinned evidence"
```

---

### Task 15: Full verification sweep

**Files:** none modified except as fixes require.

**Interfaces:**
- Consumes: everything.
- Produces: the evidence that the branch is actually done.

Run every command and paste its real output. A failing command is a task to fix, not a line to soften. Do not claim "verified" on a spot check.

- [ ] **Step 1: Per-module tests and vet**

Changed modules: `libs/atlas-constants`, `libs/atlas-packet`, `libs/atlas-saga`, `services/atlas-asset-expiration/atlas.com/asset-expiration`, `services/atlas-data/atlas.com/data`, `services/atlas-pets/atlas.com/pets`, `services/atlas-inventory/atlas.com/inventory`, `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator`, `services/atlas-channel/atlas.com/channel`.

For each, run:

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: clean in every one.

- [ ] **Step 2: Repo-root guards**

From the worktree root:

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. Run `tools/lint.sh` (no flags) first to fix formatting in place, then re-run `--check`. `lint.sh --check` false-fails without nvm on PATH — if it errors before linting anything, load nvm and re-run rather than treating it as a real failure.

`tools/service-registration-guard.sh` is not required: this task adds no service, and touches none of `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work` or `tools/db-bootstrap.sh`.

The trade / mist / npc-shop contract-mirror guards are likewise not applicable. The two mirrors this task *does* create (`ResetPetExpirationCommandBody` across atlas-pets/atlas-inventory, and the pet status events across atlas-pets/atlas-channel/atlas-saga-orchestrator) have no guard script; the Task 10 Step 7 `diff` is the manual stand-in. Re-run it here.

- [ ] **Step 3: Docker bake for every service whose `go.mod` moved**

Run from the worktree root:

```bash
git diff --name-only main... | grep 'go\.mod$' || echo "no go.mod changes"
```

If the plan was followed, this prints `no go.mod changes` and no bake is required. If any `go.mod` did move, run `docker buildx bake atlas-<svc>` for each affected service and show it clean — `go build` against `go.work` will not catch a missing `COPY libs/...` line in the shared `Dockerfile`.

- [ ] **Step 4: Cross-service seam trace**

Green unit tests on a stubbed seam are zero coverage. Confirm by reading the real consumers — not by re-running the unit tests — that:

1. The `RESET_PET_EXPIRATION` message atlas-pets buffers is decoded by atlas-inventory's `handleResetPetExpirationCommand`: same topic constant (`COMMAND_TOPIC_COMPARTMENT`), same `Type` string, and a field-for-field identical body struct (the Task 10 Step 7 diff).
2. The asset `UPDATED` event atlas-inventory emits reaches `services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go`'s `UPDATED` arm, which re-announces the slot with `invpkt.NewAddEntry` — the full slot replacement that carries the new `dateExpire` to `CUIToolTip::GetPetDeadDate`. Read the arm and confirm a pet asset is not filtered out before the announce.
3. The `REVIVED` / `REVIVE_FAILED` events atlas-pets emits are decoded by the orchestrator's consumer with matching `Type` strings and body field names.

- [ ] **Step 5: Code review before PR**

Run `superpowers:requesting-code-review`, which dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` (no frontend files changed). Pin the reviewer subagents to the standard model, not an expensive one. Findings land in `docs/tasks/task-228-water-of-life-pet-revive/audit.md`. Address them before opening the PR.

- [ ] **Step 6: Commit any fixes**

```bash
git add -A
git commit -m "chore(task-228): verification fixes"
```

---

## Acceptance Criteria Coverage

| PRD criterion | Task |
|---|---|
| Expired pet not deleted, stays in its slot | 2 |
| Expired non-pet still expires and is removed | 2 |
| Expired pet cannot be summoned | 7 |
| `WATER_OF_LIFE` decodes an empty body and round-trips on all five versions | 4, 14 |
| Handler registered in five templates, absent from the `n-a` ones | 5 |
| Template guards clean | 5, 15 |
| Revive consumes exactly one water and sets both expirations to `now + 90 days` | 9, 10, 15 (seam trace) |
| Name / level / closeness / fullness unchanged | 9 |
| Revived pet unspawned, then summonable | 9 (slot untouched), 7 (gate no longer fires) |
| Two dolls: the later past expiration wins | 12 |
| Expiration derived from WZ `life`, not a constant | 3, 9 (the differing-`life` test) |
| No dried-up pet: nothing consumed, message delivered | 12 |
| No Water of Life held: nothing changes, message delivered | 12 |
| `life` zero or absent: rejected without consumption | 12 (pre-flight), 9 and 10 (authoritative) |
| Forged expiration beyond the cap rejected, item refunded | 10 (reject), 11 (compensate) |
| Tooltip changes without relog | 15 Step 4.2 |
| Consumed water disappears in the same interaction | 12 (saga step 1's `DELETED`) |
| Five cells `✅`, `n-a` cells still pass the gate | 14 |
| Tests, vet, bake, guards clean | 15 |
