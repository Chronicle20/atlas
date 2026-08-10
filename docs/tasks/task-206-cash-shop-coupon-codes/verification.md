# task-206 — verification gate (plan Task 30)

Branch: `task-206-cash-shop-coupon-codes`. Every command below was executed from
the worktree root and its output read. Nothing here is reported from memory.

**Scope of this record.** Steps 1–6 and 9 were run. **Step 7 (live client
end-to-end) was NOT run** — it needs a human at a game client; the checklist is
parked below, unfilled. **Step 8 (code review)** is run separately by the
controller. Step 10 (committing this record) is likewise the controller's.

---

## Summary

| Step | Gate | Result |
|---|---|---|
| 1 | `go build` / `go vet` / `go test -race` in each changed module | **PASS** (after re-run with `-count=1` — see below) |
| 2 | Eight repo-root guards | **PASS** |
| 3 | `tools/lint.sh` fix mode + `--check` | **PASS after fixes** — fix mode rewrote files AND four real lint findings had to be fixed by hand |
| 4 | `packet-audit` matrix / operations / fname-doc / dispatcher-lint | **PASS after a fix** — `dispatcher-lint` FAILED first; fixed in `dc6c2fba5` |
| 5 | `docker buildx bake atlas-cashshop atlas-channel atlas-configurations` | **PASS** |
| 6 | `atlas-ui` `npm run build` + `npm test` | **PASS** |
| 7 | Live v83 client end-to-end | **NOT RUN — pending a human** |
| 9 | No previously-verified cell changed wire behaviour | **PASS** — four declared categories, nothing outside them |

Two gates were **found red and fixed on this branch**. Both are called out in
full below rather than folded into a green summary line.

### Commits this gate produced

| SHA | Subject | Why |
|---|---|---|
| `dc6c2fba5` | `fix(packet-audit): scope the FAM-CAP guard to clientbound dispatcher families` | `dispatcher-lint` was failing (Step 4) |
| `dcb94b035` | `style(task-206): satisfy tools/lint.sh over the coupon code` | fix-mode rewrites + six hand fixes (Step 3) |
| `ce8c8dbe6` | `chore(packets): regenerate the matrix toolSha after the FAM-CAP fix` | `matrix --check` toolSha drift caused by `dc6c2fba5` |

### Final-tree re-run

Because those three commits changed Go source and the packet tool **after**
Steps 1/2/4/5 first ran, all four were **re-run in full against the final tree**
(`ce8c8dbe6`). The results recorded below are the final-tree results, not the
first-pass ones.

| Gate | Final-tree result |
|---|---|
| Step 1, six modules, `-count=1 -race` | all exit 0, 0 FAIL lines |
| Step 2, eight guards | all exit 0 |
| Step 4, `dispatcher-lint` / `operations --check` / `fname-doc --check` / `doc-freshness --check` / `gate-check --check` / `matrix --check` | all exit 0 |
| Step 4, `matrix` regen | no diff |
| Step 5, `docker buildx bake atlas-cashshop atlas-channel atlas-configurations` | `BAKE2_EXIT=0`, all three images named |

Working tree after the re-run is clean apart from this untracked file.

---

## Step 1 — Go module gates

The changed-module list was derived from the diff rather than trusted from the
brief:

```
$ git diff --name-only main...HEAD | grep -E '\.go$|go\.(mod|sum)$' | awk -F/ '{print $1"/"$2}' | sort -u
libs/atlas-constants
libs/atlas-packet
libs/atlas-redis
services/atlas-cashshop
services/atlas-channel
tools/packet-audit
```

This **differs from the brief's list in two ways**, and the diff is
authoritative:

- `libs/atlas-tenant` is **not** changed (the brief listed it conditionally;
  nothing under it moved, so it was dropped as the brief allows).
- `libs/atlas-constants` **is** changed (the shared coupon `Normalize` /
  `Plausible` helpers live there by a human ruling rather than being duplicated
  per service) and the brief's list omits it. It was gated like the rest.

Only one `go.mod` was touched:

```
$ git diff --name-only main...HEAD | grep -E 'go\.(mod|sum)$'
services/atlas-cashshop/atlas.com/cashshop/go.mod
services/atlas-cashshop/atlas.com/cashshop/go.sum
```

Command, per module (service modules live at
`services/atlas-<svc>/atlas.com/<svc>`, not at the service root — running it at
the service root fails with *"directory prefix . does not contain modules listed
in go.work"*):

```
$ ( cd "$m" && go build ./... && go vet ./... && go test -count=1 -race ./... )
```

| Module | exit | `ok` pkgs | FAIL lines |
|---|---|---|---|
| `libs/atlas-packet` | 0 | 80 | 0 |
| `libs/atlas-redis` | 0 | 1 | 0 |
| `libs/atlas-constants` | 0 | 11 | 0 |
| `tools/packet-audit` | 0 | 13 | 0 |
| `services/atlas-cashshop/atlas.com/cashshop` | 0 | 13 | 0 |
| `services/atlas-channel/atlas.com/channel` | 0 | 110 | 0 |

**PASS.**

### A false green on the first pass — read this

The first pass ran `go test -race ./...` **without `-count=1`** and reported
`tools/packet-audit` as `ok ... (cached)`. That was a **stale-cache false
green**: `TestFamilyCapRealTreeClean` was in fact failing on this branch. Go's
test cache does not invalidate on a `readdir` of a directory outside the package
under test, and that test walks `docs/packets/dispatchers/` — into which this
branch added two files.

```
$ cd tools/packet-audit && go test -count=1 -run TestFamilyCap ./cmd/
--- FAIL: TestFamilyCapRealTreeClean (0.01s)
    family_cap_test.go:97: real tree must pass family-cap guard; got [
      ../../../docs/packets/dispatchers/cash_shop_coupon_code.yaml	FAM-CAP	family CCashShop::OnStatusCoupon is neither discrete-implemented in run.go ...
      ../../../docs/packets/dispatchers/cash_shop_operation_handle.yaml	FAM-CAP	family CCashShop::RequestCashPurchaseRecord is neither discrete-implemented in run.go ...]
FAIL
```

The table above is from the **`-count=1` re-run**, after the fix. Any future
run of this gate on this repo must pass `-count=1`.

---

## Step 2 — Repo-root guards

Each script was run from the worktree root and its exit code read.

| Guard | exit | Tail of output |
|---|---|---|
| `tools/redis-key-guard.sh` | 0 | per-module scan lines, no violations |
| `tools/goroutine-guard.sh` | 0 | per-module scan lines, no violations |
| `tools/skill-job-id-guard.sh` | 0 | `skill-job-id-guard: clean (14 divergent const(s) checked)` |
| `tools/buff-duration-guard.sh` | 0 | per-module scan lines, no violations |
| `tools/template-opcode-order-guard.sh` | 0 | `OK: 22 template arrays are in ascending opcode order.` |
| `tools/template-duplicate-binding-guard.sh` | 0 | `OK: 22 template arrays carry no duplicate (name, opCode) binding.` |
| `tools/template-movement-types-guard.sh` | 0 | `template_gms_84_1.json: 5 move handler(s)` … (5 per template) |
| `tools/service-registration-guard.sh` | 0 | `service-registration-guard: clean` |

**PASS.** The opcode-order and duplicate-binding guards matter here because this
branch inserts a `CashShopCouponCodeHandle` entry into ten templates;
`service-registration-guard.sh` is green because no service registration changed,
which is what it is here to prove.

---

## Step 3 — Lint

`tools/lint.sh` needs Node 22 for its `atlas-ui` leg, so each invocation was
preceded by sourcing nvm and `nvm use 22` (`node --version` → `v22.22.2`).

### Fix mode, first run — FAILED, with two distinct outcomes

```
$ tools/lint.sh
...
lint.sh: FAIL — 2 failing target(s):
lint.sh:   lint:services/atlas-cashshop/atlas.com/cashshop
lint.sh:   lint:services/atlas-channel/atlas.com/channel
LINT_FIX_EXIT=1
```

**(a) Fix mode rewrote four files in place.** This is a real change that needs
committing, not a no-op:

```
$ git diff --stat
 services/atlas-cashshop/atlas.com/cashshop/wallet/model.go          |  7 ++--
 services/atlas-ui/src/lib/hooks/api/__tests__/useCoupons.test.tsx   | 38 ++++++---
 services/atlas-ui/src/services/api/__tests__/coupons.service.test.ts| 10 ++--
 services/atlas-ui/src/services/api/coupons.service.ts               |  8 +--
```

The Go rewrite is a gofumpt/staticcheck `if/else if/else` → `switch` conversion
in `wallet/model.go`'s `Award`:

```go
-	if currency == 1 {
+	switch currency {
+	case 1:
 		newCredit = add(newCredit)
-	} else if currency == 2 {
+	case 2:
 		newPoints = add(newPoints)
-	} else {
+	default:
 		newPrepaid = add(newPrepaid)
 	}
```

The three `atlas-ui` rewrites are Prettier reformatting of this branch's new
coupon service + tests. All four are behaviour-preserving.

**(b) Four lint findings fix mode could not auto-fix — all in this branch's new
code**, all fixed by hand:

| File:line | Linter | Finding | Fix |
|---|---|---|---|
| `services/atlas-cashshop/.../coupon/migration_test.go:50` | errcheck | `rows.Close` return unchecked | `defer func() { _ = rows.Close() }()` |
| `services/atlas-cashshop/.../coupon/migration_test.go:70` | errcheck | `colRows.Close` return unchecked | same idiom |
| `services/atlas-cashshop/.../coupon/resource_test.go:101` | errcheck | `resp.Body.Close` return unchecked | same idiom |
| `services/atlas-cashshop/.../coupon/redemption/provider.go:36` | unused | `countByCouponIdProvider` is dead | **deleted**, replaced by a comment explaining why no such provider exists |
| `services/atlas-channel/.../kafka/consumer/cashshop/consumer.go:205` | staticcheck S1023 | redundant `return` | removed |
| `services/atlas-channel/.../kafka/consumer/cashshop/consumer.go:226` | staticcheck S1023 | redundant `return` | removed |

`countByCouponIdProvider` was genuinely dead, not merely not-yet-wired: the
per-coupon redemption count is **materialized on the coupon row**
(`coupon.Model.redemptionCount`, incremented atomically by `reserveUse`) and
served from there by `coupon.RestModel.RedemptionCount`
(`coupon/rest.go:28,67`). Counting the redemption table would have been a
second, racier source of the same number, so the provider was removed rather
than wired up.

Note: the first fix-mode log also carries `level=warning` lines in which
golangci-lint resolves paths into a **sibling worktree**
(`.worktrees/task-204-dual-blade-job-tree/...`, "no such file or directory").
That is cross-worktree noise from the shared Go workspace, not a finding against
this branch, and it did not affect any target's pass/fail.

### Re-run after the fixes

```
$ tools/lint.sh            # fix mode
✖ 7 problems (0 errors, 7 warnings)
lint.sh: OK
LINT_FIX2_EXIT=0

$ git status --short        # fix mode rewrote nothing new — idempotent
(only the same files already rewritten above, plus the hand fixes)

$ tools/lint.sh --check
✖ 7 problems (0 errors, 7 warnings)
lint.sh: OK
LINT_CHECK_EXIT=0
```

The 7 ESLint items are **warnings, 0 errors**, and are pre-existing
(`react-hooks/exhaustive-deps` and `react-hooks/incompatible-library` in
`AccountsPage.tsx`, `QuestsPage.tsx` and a dialog using
`react-hook-form`'s `watch()`); none is in this branch's new coupon code and
they do not fail the gate.

**PASS.** No `ui:node-missing` failure and no lock-contention failure was hit on
the passing run. Both the fix-mode rewrites and the six hand fixes are committed
in `dcb94b035`.

---

## Step 4 — Packet-audit gates

```
$ go run ./tools/packet-audit matrix
note	n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
wrote docs/packets/audits/STATUS.md and docs/packets/audits/status.json
exit=0

$ git diff --stat docs/packets/audits
(no output — regenerating the matrix produced no diff, so the committed
 STATUS.md / status.json are current)

$ go run ./tools/packet-audit operations --check
operations check OK (0 absent-writer note(s))
exit=0

$ go run ./tools/packet-audit fname-doc --check
fname-doc check OK (252 structs without an audit report carry no fname)
exit=0
```

The other CI gates in `.github/workflows/packet-matrix.yml` were run too, since
this branch modifies the tool:

```
$ go run ./tools/packet-audit doc-freshness --check
doc-freshness: PROCESS.md packet-process-facts agree with the tool (10 versions, 7 CI gates).
exit=0

$ go run ./tools/packet-audit gate-check --check
gate-check: all 19 gate(s) have verified byte-fixtures on both straddling versions (0 partial-by-design).
exit=0

$ go run ./tools/packet-audit matrix --check
exit=0
```

**`matrix --check` briefly went red on the final-tree re-run**, and the fix is
recorded rather than glossed. `dc6c2fba5` changed `tools/packet-audit`, and the
matrix records a `toolSha` derived from that source tree, so the committed
matrix went stale. Regenerating produced a **two-line diff and nothing else**:

```
-Tool: `ec163ac2ab5dcb6d25ba9adba2958c46697409bd0c1e1bd247bc72c2d937c7df`
+Tool: `1bde9e1f1dfb0f5f970bc29aa898b9be00a829d48f604db3ee087f284c69caa8`
```

Zero coverage cells changed state and every export hash is unchanged. Committed
as `ce8c8dbe6`; `matrix --check` exits 0 again.

### `dispatcher-lint` FAILED, and had to be fixed

```
$ go run ./tools/packet-audit dispatcher-lint
docs/packets/dispatchers/cash_shop_coupon_code.yaml	FAM-CAP	family CCashShop::OnStatusCoupon is neither discrete-implemented in run.go (no #-suffixed arms) nor listed in families.yaml/baseline — author it discrete-per-mode or add a baseline/families cap [family=CCashShop::OnStatusCoupon]
docs/packets/dispatchers/cash_shop_operation_handle.yaml	FAM-CAP	family CCashShop::RequestCashPurchaseRecord is neither discrete-implemented in run.go (no #-suffixed arms) nor listed in families.yaml/baseline — author it discrete-per-mode or add a baseline/families cap [family=CCashShop::RequestCashPurchaseRecord]
packet-audit dispatcher-lint: 2 violation(s) (see docs/packets/DISPATCHER_FAMILY.md)
exit status 1
```

Both violations are on files **this branch added**, so this was a regression
introduced here, not a pre-existing red. Both are false positives:

- **Mode-prefix capping is a clientbound concept.**
  `docs/packets/evidence/families.yaml` (consumed by `matrix` `grade.go`) exists
  because a byte fixture aimed at a client `switch(Decode1())` demultiplexer
  proves only the one arm it exercises. A **serverbound**
  `docs/packets/dispatchers/*.yaml` is not that shape — it is the source of
  truth for a *handler's* `options.operations` **routing** table.
- `cash_shop_coupon_code.yaml` has **no `operations:` section at all** — a
  single serverbound op with one struct
  (`libs/atlas-packet/cash/serverbound/coupon_code.go:44` carries
  `// packet-audit:fname CCashShop::OnStatusCoupon`). It cannot be a
  demultiplexer.
- `cash_shop_operation_handle.yaml`'s modes are written by **N independent
  client call sites** — the file says so itself ("The client has NO single
  cash-shop request builder… The 'mode switch' is a SERVER-side construct"), and
  the ServerBound registry lists **25 distinct FNames** for
  `CASHSHOP_OPERATION`, each already a discrete struct with its own
  `packet-audit:fname` in `libs/atlas-packet/cash/serverbound/shop_operation_*.go`.

Neither disposition FAM-CAP offers was correct:

- **`families.yaml`** would have been actively harmful. `grade.go` applies
  `in.Families[baseFName(ref.FName)]` **unconditionally per-op**, and the
  registry fnames for these two ops are exactly `CCashShop::OnStatusCoupon` and
  `CCashShop::RequestCashPurchaseRecord`. An entry would have demoted the 10
  `COUPON_CODE` and 7 `CASHSHOP_OPERATION` serverbound cells that already carry
  byte fixtures from ✅ back to 🧩. `run.go` records the same reasoning for the
  task-145 `OnClaimResult` / `OnSueCharacterResult` precedent.
- **The baseline** (`docs/packets/dispatcher-lint-baseline.yaml`) is
  `exempt_families: []`, legacy-only, and documented as only-ever-shrinking.

**Fix (commit `dc6c2fba5`)**: `checkFamilyCap` now skips files declaring
`direction: serverbound`, with the rationale recorded in the code. Clientbound
files keep the full guard. A regression test
(`TestFamilyCapServerboundSkipped`) pins that a serverbound file is skipped
while a clientbound phantom in the same directory still fails. The one
pre-existing serverbound file, `character_interaction_handle.yaml`, previously
passed only by fname collision with its clientbound sibling
(`character_interaction.yaml`, same `CMiniRoomBaseDlg::OnPacketBase`); it now
passes on purpose.

```
$ go run ./tools/packet-audit dispatcher-lint
dispatcher-lint: clean
exit=0
```

**PASS after the fix.**

### Coverage row

```
$ python3 -c "... status.json, op == COUPON_CODE ..."
gms_v48  verified 161
gms_v61  verified 197
gms_v72  verified 220
gms_v79  verified 222
gms_v83  verified 230
gms_v84  verified 236
gms_v87  verified 243
gms_v92  verified 269
gms_v95  verified 276
jms_v185 verified 246
```

All ten cells verified, and `gms_v84` sits at **236**, which is the corrected
opcode. The brief's expectation that `gms_v48`/`v61`/`v72`/`v79` would read
`n-a` is superseded: Task 3 (`bd2827f7c`) proved all four legacy clients send
the op and Task 4 (`ec33e4b0f`) shipped the codec with byte fixtures, so those
four are `verified` rather than `n-a` — the brief's "unless Task 3 promoted one"
branch fired for all four.

---

## Step 5 — Docker bake (mandatory)

```
$ docker buildx bake atlas-cashshop atlas-channel atlas-configurations
...
#196 naming to docker.io/library/atlas-channel:local done
#196 unpacking to docker.io/library/atlas-channel:local 0.3s done
#196 DONE 2.4s
BAKE_EXIT=0
```

**PASS.** All three images built. This is the step that catches a missing
`COPY libs/...` in the shared root `Dockerfile`, which `go build` against
`go.work` cannot; `services/atlas-cashshop`'s `go.mod` gained `libs/atlas-redis`
on this branch. Both required lines are present and were exercised by the build:

```
$ grep -n "COPY libs/atlas-redis\|COPY libs/atlas-constants" Dockerfile
32:COPY libs/atlas-constants/go.mod   libs/atlas-constants/go.sum   libs/atlas-constants/
41:COPY libs/atlas-redis/go.mod       libs/atlas-redis/go.sum       libs/atlas-redis/
62:COPY libs/atlas-constants   libs/atlas-constants
71:COPY libs/atlas-redis       libs/atlas-redis
```

`atlas-saga-orchestrator` is deliberately absent — design §2 removed it from the
blast radius.

---

## Step 6 — `atlas-ui`

```
$ cd services/atlas-ui && npm run build && npm test
```

```
✓ built in 1.19s
UI_BUILD_EXIT=0

 RUN  v4.1.10 .../services/atlas-ui
 Test Files  246 passed (246)
      Tests  2003 passed (2003)
   Duration  39.20s
UI_TEST_EXIT=0
```

**PASS.** Both legs run, not just `npm test` — `npm run build` type-checks the
tests, so a green `npm test` alone would not have been sufficient. Node 22 was
active (`v22.22.2`). The build emits a chunk-size advisory for
`ConversationEditorPanel` (1.65 MB); it is a warning, pre-existing, and
unrelated to the coupon work.

---

## Step 7 — Live v83 client end-to-end — **NOT RUN**

**This step was not attempted and no result is claimed for it.** It requires a
human driving an actual game client against an ephemeral deployment of this
branch. It is the single most valuable evidence this task can produce, because
most `errors` key **names** are ordinal alignment against the decompiled enum
rather than observed text — a live observation converts an `aligned` row into a
`verified` one.

Checklist, to be filled in from the live run:

| # | Scenario | Expected | Observed | Pass? |
|---|---|---|---|---|
| 1 | Mint a code via the UI with a Maple Points reward **and** a cash-item reward | code created, both rewards attached | | |
| 2 | Cash Shop → Coupon tab → enter the code | both rewards land; balance and locker update **without a relog** | | |
| 3 | Enter the same code again | "already used" | | |
| 4 | Enter a garbage code | "invalid code" | | |
| 5 | Enter an expired code | "expired" | | |
| 6 | Enter a code whose global uses are exhausted | "usage limit" | | |

Attach pod logs for the two that are easy to get wrong: the delta-vs-balance
rendering in scenario 2, and the arm selection in scenarios 3–6.

**If any message renders as the generic default notice instead of the specific
text**, that is an `aligned` error row whose ordinal alignment was wrong: record
the observed byte and the observed text, and correct both `derivation.md` and
the YAML from the observation.

---

## Step 9 — No previously-verified cell changed wire behaviour

Two independent sweeps: the codec diff and a **mechanical numeric comparison**
of every scalar leaf of every changed tenant template.

### 9a — `libs/atlas-packet`

Only three files moved, and only one of them already existed:

```
$ git diff --stat main...HEAD -- libs/atlas-packet
 libs/atlas-packet/cash/clientbound/shop_operation_body.go |   2 +-
 libs/atlas-packet/cash/serverbound/coupon_code.go         | 127 ++++++++
 libs/atlas-packet/cash/serverbound/coupon_code_test.go    | 352 +++++++++++++++
 3 files changed, 480 insertions(+), 1 deletion(-)
```

`coupon_code.go` / `coupon_code_test.go` are the **new** codec — no prior cell.
The single line in the pre-existing `shop_operation_body.go` is the key-string
correction, not a wire change (see category 3).

### 9b — Tenant templates: mechanical numeric comparison

Every changed template was parsed as JSON at `main` and at `HEAD`, its
`handlers`/`writers` arrays re-keyed by `(name, opCode)` so index shifts do not
register, and **every scalar leaf compared numerically** (hex strings, decimal
strings and JSON numbers all normalised to a number before comparing):

```
=== template_gms_12_1.json   added=2  removed=0 repr-only=0  NUMERIC-CHANGED=0
=== template_gms_48_1.json   added=47 removed=0 repr-only=0  NUMERIC-CHANGED=0
=== template_gms_61_1.json   added=50 removed=0 repr-only=5  NUMERIC-CHANGED=0
=== template_gms_72_1.json   added=56 removed=0 repr-only=14 NUMERIC-CHANGED=0
=== template_gms_79_1.json   added=60 removed=0 repr-only=14 NUMERIC-CHANGED=0
=== template_gms_83_1.json   added=60 removed=0 repr-only=14 NUMERIC-CHANGED=1
   NUMCHG /socket/handlers/{handler=CashShopOperationHandle}/options/operations/BUY_NORMAL: 20 -> 32
=== template_gms_84_1.json   added=60 removed=0 repr-only=14 NUMERIC-CHANGED=1
   NUMCHG /socket/handlers/{handler=CashShopOperationHandle}/options/operations/BUY_NORMAL: 20 -> 32
=== template_gms_87_1.json   added=79 removed=0 repr-only=14 NUMERIC-CHANGED=0
=== template_gms_92_1.json   added=77 removed=0 repr-only=14 NUMERIC-CHANGED=0
=== template_gms_95_1.json   added=77 removed=0 repr-only=14 NUMERIC-CHANGED=0
=== template_jms_185_1.json  added=37 removed=0 repr-only=0  NUMERIC-CHANGED=0

TOTALS added=605 removed=0 repr-only=103 NUMERIC-CHANGED=2
```

**Zero keys were removed from any template**, and exactly **two** scalar values
changed numerically across the whole branch. All 605 additions classify cleanly:

```
476  new options.errors table entries
 57  new serverbound options.operations keys (v87/v92/v95, 19 each)
 50  new CashShopCouponCodeHandle entries (10 templates × 5 fields)
 22  new /cashShop/coupons/rateLimit/{attempts,windowSeconds} (11 templates × 2)
```

### 9c — The four declared categories

| # | Category | Extent | Wire-affecting? | Evidence |
|---|---|---|---|---|
| 1 | **New `CouponCode` codec + new `CashShopCouponCodeHandle` template entries** | `libs/atlas-packet/cash/serverbound/coupon_code.go`; 50 leaves across **ten** templates (all but `gms_12`, which has no cash shop) | New behaviour — **no prior cell**, so nothing to regress | `docs/packets/dispatchers/cash_shop_coupon_code.yaml`; ten `docs/packets/evidence/*/cash.serverbound.CouponCode.yaml` records; `derivation.md` §"Legacy versions — COUPON_CODE applicability" (:1909). Ten templates, not six: Task 3 (`bd2827f7c`) proved `gms_v48/61/72/79` all send the op, so those are new-behaviour cells with no prior verified state |
| 2 | **New `options.errors` tables** | 476 leaves across all 11 templates | Prior behaviour was `ResolveCode` returning **99** — a broken cell, not a verified one | `derivation.md` per-version "### `errors` enum" sections (:100, :406, :556, :686, :1007, :1173, :1361, :1501, :1641, :1781) |
| 3 | **Evidenced corrections** | 4 items — see the table below | Yes, deliberately | see below |
| 4 | **Hex-string → integer normalization by `packet-audit operations` write mode** | 103 leaves across **8** templates | **No — provably representation-only** | see below |

#### Category 3 — every deliberate wire-affecting correction

| Correction | Where | Evidence address |
|---|---|---|
| `INVALID_COUPON_COUPON` → `INVALID_COUPON_CODE` | `libs/atlas-packet/cash/clientbound/shop_operation_body.go:91` | The constant was **unconsumed on `main`** — `git grep CashShopOperationErrorInvalidCouponCode main -- libs services` returns only its own definition line. No shipped byte or key string changed for any existing caller; the corrected spelling is what the new coupon arm and the templates now agree on |
| `gms_v84` `COUPON_CODE` opcode 230 → **236** | `docs/packets/registry/gms_v84.yaml` | `derivation.md:355` "### Registry bug — `COUPON_CODE` opcode" and :360 — `COutPacket(&pkt, 236)` @ `0x473c84` inside `CCashShop::OnStatusCoupon` @ `0x473bde`. 230 was a csv-import seed off the v83 column (the CSVs have no v84 column); the `68 E6 00 00 00` hit at `0x473fd1` is a `CreateDlg` pixel coordinate in `sub_473F02`, not a packet ctor. 236 = `CASHSHOP_OPERATION`(235) + 1, matching every other version's +1 relation. The cell was never verified at 230, so no verified cell regressed |
| `BUY_NORMAL` 20 → **32** on `gms_v83` and `gms_v84` | `template_gms_83_1.json`, `template_gms_84_1.json`, `CashShopOperationHandle.options.operations` | `derivation.md:82` (v83: `CCashShop::OnBuyNormal` ctor @ `0x46e79e`, `push 20h` @ `0x46e7ab`) and `derivation.md:387` (v84: ctor @ `0x471227`, `push 20h` @ `0x471234`). `20h` = 32; the template's `20` was a hex-digits-read-as-decimal transcription error, documented at `derivation.md:89` "**Template bug — `BUY_NORMAL`**" and in the `cash_shop_operation_handle.yaml` inline comment. This is a **fix to a wrong byte**, i.e. the old value could never have worked |
| New serverbound `operations` tables on `gms_v87` / `gms_v92` / `gms_v95` | 19 keys each, 57 total | Additions, not changes — those three versions had **no** serverbound `CashShopOperationHandle.options.operations` table at all before. `derivation.md:526` (v87), :790 (v92), :1049 (v95); enumerated by exhaustive byte search for the opcode ctor push across the `CCashShop` region, so complete by construction |

#### Category 4 — the `packet-audit operations` write-mode normalization (declared and cleared)

Running `packet-audit operations` in **write** mode rewrote hex-string operation
values as integers. The mechanical sweep above reports **103 such leaves across
8 templates** (`gms_61`×5, `gms_72`/`79`/`83`/`84`/`87`/`92`/`95`×14), and
**every one has `NUMERIC-CHANGED=0`** — the normalised numeric value is
identical on both sides. All 103 are in exactly two clientbound writers:

```
writers/{writer=ClaimResult}/options/operations/CANNOT_CONNECT:     '0x44' -> 68
writers/{writer=ClaimResult}/options/operations/EXCEEDED:           '0x45' -> 69
writers/{writer=ClaimResult}/options/operations/FALSE_REPORT_CITED: '0x48' -> 72
writers/{writer=ClaimResult}/options/operations/NOT_ENOUGH_MESOS:   '0x43' -> 67
writers/{writer=ClaimResult}/options/operations/RECHECK_NAME:       '0x42' -> 66
writers/{writer=ClaimResult}/options/operations/REPORTED_NOTICE:    '0x03' -> 3
writers/{writer=ClaimResult}/options/operations/SUCCESS:            '0x02' -> 2
writers/{writer=ClaimResult}/options/operations/TIME_WINDOW:        '0x47' -> 71
writers/{writer=ClaimResult}/options/operations/TRY_AGAIN:          '0x41' -> 65
writers/{writer=SueCharacterResult}/options/operations/DAILY_LIMIT:      '0x02' -> 2
writers/{writer=SueCharacterResult}/options/operations/GENERIC_FAILURE:  '0x04' -> 4
writers/{writer=SueCharacterResult}/options/operations/REPORTED_NOTICE:  '0x03' -> 3
writers/{writer=SueCharacterResult}/options/operations/SUCCESS:          '0x00' -> 0
writers/{writer=SueCharacterResult}/options/operations/UNABLE_TO_LOCATE: '0x01' -> 1
```

It is provably not wire-affecting because `ResolveCode` accepts both the
`float64` and the `string` form, so both representations resolve to the same
byte. **Declared and cleared — a reviewer does not need to re-audit these 103
leaves.**

There is a **fifth, non-wire** class the plan's three categories do not name,
recorded here rather than left as an unexplained residue: the 22 leaves
`/cashShop/coupons/rateLimit/{attempts,windowSeconds}` (11 templates × 2). These
are new **tenant configuration** for the coupon rate limiter (PRD §8 rate
limiting, design §7.5, plan Tasks 10/17). They are not part of any packet and
change no wire byte. `gms_12` receives only these two — it has no cash shop and
correctly gets no coupon handler and no errors table.

### 9d — The `gms_v92` clientbound column changed zero template keys

Confirmed independently by the mechanical sweep: on `template_gms_92_1.json` the
`CashShopOperation` clientbound writer's `operations` map has **zero added and
zero changed keys** — the file's only `repr-only` leaves are the 14
`ClaimResult`/`SueCharacterResult` ones above, and its 77 additions are all
coupon handler (5) + errors (51) + serverbound operations (19) + rate limit (2).
This matches `derivation.md:983`: *"Diff against `template_gms_92_1.json` (57
existing keys): ZERO differences… no v92 wire byte changed."* Establishing the
v92 clientbound column in the YAML only means `--check` now validates those 57
keys instead of skipping them.

### 9e — Verdict

**Nothing outside the four declared categories (plus the declared non-wire
rate-limit config).** Zero template keys removed; exactly two numeric changes,
both the evidenced `BUY_NORMAL` correction.

---

## Gates NOT covered by this record

- **Step 7** — live client end-to-end. Not run; pending a human. See above.
- **Step 8** — code review (`plan-adherence-reviewer`,
  `backend-guidelines-reviewer`, `frontend-guidelines-reviewer`,
  `packet-completeness-critic`). Run separately by the controller.
- **Step 10** — committing this record. Left to the controller.
