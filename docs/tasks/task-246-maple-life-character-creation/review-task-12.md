# Review — Task 12: the Maple Life duplicate-name probe handler

Range: `e3a8953a2..6673126` (single commit `6673126f4`).

Inputs consulted: `.superpowers/sdd/plan/task-12-brief.md` (body + "Controller
addendum (Task 12)"), `.superpowers/sdd/plan/task-12-report.md`, and the
retraction described in
`docs/tasks/task-246-maple-life-character-creation/bug-543-is-the-submit-not-the-open.md`
(addendum §5 and `TestMapleLifeCheckNameWithoutPendingRecordIsRejected`
retracted; unconditional validity check, no registry lookup).

## 1. Retraction applied cleanly — PASS

`git diff e3a8953a2..6673126 | grep -n "GetRegistry\|maplelife\.Get\|PhaseOpen\|registry"`
returns only two doc-comment lines
(`maple_life_check_name.go:93,98`) explaining *why* the registry is not
consulted — no executable reference remains.

- No `maplelife.GetRegistry()` / `.Get(...)` call anywhere in
  `maple_life_check_name.go` or `maple_life_check_name_test.go`.
- No `atlas-channel/maplelife` or `atlas-tenant`-for-registry-purposes import
  survives in the handler (`maple_life_check_name.go:1-16` imports:
  `atlas-channel/character`, `atlas-channel/session`,
  `atlas-channel/socket/writer`, `context`, `logrus`, `atlas-constants/world`,
  `maplelife/clientbound`, `maplelife/serverbound`, `atlas-socket/packet`,
  `atlas-socket/request` — all used).
- Test file's `tenant` import (`maple_life_check_name_test.go:23`) is only
  used to build the tenant context for `mustTenant`/`WithContext`, not to seed
  a registry entry — no `openDialog()` helper, no `PhaseOpen` fixture,
  present anywhere (`grep -c PhaseOpen` over both files = 0).
- `git diff e3a8953a2..6673126 -- services/atlas-channel/atlas.com/channel/maplelife/`
  is empty — registry package itself untouched, confirming "read-only" (now
  moot) was honored to the letter of "don't touch."
- `git diff --stat` shows only 3 files touched (handler, handler test,
  `main.go`) — no orphaned fixture elsewhere in the module.

Clean. No residue found.

## 2. FR-5.4 survived the retraction — PASS

`handleMapleLifeCheckName` (`maple_life_check_name.go:74-102`) has exactly
three exit points, each writing one `MAPLELIFE_RESULT` via
`announceMapleLifeResult` and then returning (or falling off the end, which is
the same as returning since it's the last statement):

- seam error (`:83-87`): `l.WithError(err).Errorf(...)` (Error level) →
  `mlcb.MapleLifeResultBody(name, mlcb.MapleLifeResultUnknownError)` → return.
- valid (`:89-92`): Debug level → `MapleLifeResultAvailable` → return.
- rejection, including unmapped reason (`:94-100`): unmapped reason logs
  `l.Errorf(...)` (Error level, `:96`); known reason logs `l.Infof(...)`
  (`:98`); both fall through to the single
  `mlcb.MapleLifeResultRejectedBody(name, res.Reason)` announce at `:100`.

No path returns without writing. Verified live: `TestMapleLifeCheckNameMapsReasons`
covers all seven cases (available/duplicate/reserved/length/regex/unknown
reason/seam error) and asserts both the written byte and, where applicable,
the log level — all pass (`go test ./socket/handler/... -run
'MapleLifeCheckName' -v`, all green, confirmed by direct re-run in this
review, not just the report's transcript).

## 3. No second reason→arm table — PASS

`maple_life_check_name.go:85,91,100` — every call site:

- `:85` `mlcb.MapleLifeResultBody(name, mlcb.MapleLifeResultUnknownError)` —
  named constant, not a reason string.
- `:91` `mlcb.MapleLifeResultBody(name, mlcb.MapleLifeResultAvailable)` — named
  constant.
- `:100` `mlcb.MapleLifeResultRejectedBody(name, res.Reason)` — the *only*
  place a raw `character.NameReason*` string reaches the codec, and it goes
  through the safe wrapper (`result.go:137-149`) whose own unmapped-key
  fallback resolves to `MapleLifeResultUnknownError`, never the raw key.

`mapleLifeKnownReasons` (`:38-43`) is a `map[string]struct{}` used only to
decide the *log level* for an unrecognised reason — it carries no arm value
and cannot drift from the codec's table, since it is consulted before the
call to `MapleLifeResultRejectedBody`, not instead of it. This matches the
report's own characterization and the addendum's ruling.

No call site ever hands a raw reason key to `MapleLifeResultBody` (the unsafe
entry point) — confirmed by grep above; the only two `MapleLifeResultBody`
calls use the two named-constant arms.

## 4. `main.go` registration — PASS

`git diff` on `main.go`:
```
+	msb "github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/serverbound"
...
+	handlerMap[msb.MapleLifeCheckNameHandle] = handler.MapleLifeCheckNameHandleFunc
```
`msb.MapleLifeCheckNameHandle` is the exported constant
(`libs/atlas-packet/maplelife/serverbound/check_name.go:13`,
`"MapleLifeCheckNameHandle"`), not a string literal. Checked all four
in-scope seed templates directly —
`services/atlas-configurations/seed-data/templates/template_gms_{83,87,92,95}_1.json`
all contain the literal `MapleLifeCheckNameHandle` handler-name string (Task
7's routing), matching exactly. `go build ./...` succeeds (import resolves,
no unused-import or redeclaration errors).

## 5. `NameScopeWorld`, asserted through the seam — PASS

`handleMapleLifeCheckName` calls
`mapleLifeNameValidityFunc(l, ctx, name, s.WorldId(), character.NameScopeWorld)`
(`:82`) — `NameScopeWorld`, not `NameScopeTenant`.

`TestMapleLifeCheckNameAsksForWorldScope`
(`maple_life_check_name_test.go:157-176`) asserts this **through the swapped
seam**: `mapleLifeNameValidityFunc` is replaced in `newMapleLifeCheckNameEnv`
(`:108-117`) with a recorder that captures `(name, worldId, scope)` into
`env.validityCalls`, and the test asserts `c.scope != character.NameScopeWorld`
and `c.worldId != env.s.WorldId()` against the *captured call*, not against a
constant re-stated in the handler source. This satisfies the doc-comment
convention the addendum cites — re-run and confirmed passing.

## 6. Exhaustiveness property — PASS, with an honest caveat already ruled on

`TestMapleLifeCheckNameReasonTableIsExhaustive`
(`maple_life_check_name_test.go:224-243`) drives `handleMapleLifeCheckName`
once per `character.NameReasonLength`, `NameReasonRegex`,
`NameReasonDuplicate`, `NameReasonReserved` and asserts each produces a
non-`AVAILABLE` arm. Confirmed passing (4/4 subtests). This is exactly the
addendum §2 replacement, not the brief's original (unwritable, per addendum,
since the codec's map is unexported).

Caveat, stated for the record and not a finding: this test would **not**
independently fail if a fifth `character.NameReason*` constant were added and
left unmapped in the codec's table, because the loop only iterates a
hardcoded literal list of the four current constants — it does not walk
`character`'s exported constants reflectively. That gap is explicitly and
correctly assigned elsewhere by the addendum: `TestMapleLifeResultReasonMapping`
(`result_test.go:231`, in `libs/atlas-packet`) owns exhaustiveness of the
codec's own map against its own literal set, and is out of this task's scope.
This is the addendum's own design, not a defect introduced by the
implementer.

## 7. Scope — PASS

`git diff --stat e3a8953a2..6673126`:
```
services/atlas-channel/atlas.com/channel/main.go   |   2 +
.../socket/handler/maple_life_check_name.go        | 109 ++++++++++
.../socket/handler/maple_life_check_name_test.go   | 241 +++++++++++++++++++++
3 files changed, 352 insertions(+)
```
Only `services/atlas-channel/atlas.com/channel` changed; nothing under
`libs/`, `docs/packets/`, seed templates, `services/atlas-login/`,
`services/atlas-character-factory/`, `services/atlas-saga-orchestrator/`, or
`deploy/`.

`git diff e3a8953a2..6673126 -- services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change.go`
is empty — byte-identical, confirming routing outcome (A) was honored (file
not touched at all, per addendum §1, stricter than the brief's "modify only
under outcome (B)").

`services/atlas-channel/atlas.com/channel/maplelife/` (the registry package)
diff is empty — unmodified.

Pre-existing `cash_shop_check_name_change_test.go` cases re-run directly in
this review (`TestNameChangeCheckReportsEveryUnavailableCause`,
`TestNameChangeCheckUsesTenantScope`) — both PASS, confirming the cash-shop
rename's `TENANT` scope assertion is unchanged and unaffected by this task's
addition of a separate `mapleLifeNameValidityFunc` seam var.

Single commit in range (`6673126f4`), matching the brief's Step 5 commit
message intent (report says the same).

## Not evaluable

None. All seven assigned checks were directly verifiable within the diff
surface plus the two read-only files (`result.go`, `check_name.go`) the diff
genuinely depends on.

## Verdict

All seven checks PASS with direct evidence. No blocking or non-blocking
findings.
