# Task 25 review — atlas-channel MAKER_SKILL handler

Commit reviewed: `7bdecda0a` ("feat(channel): handle MAKER_SKILL and forward crafts to atlas-maker")
Preceding commit `52c89dc5d` (Task 24 lint fix) out of scope, not re-reviewed.

## Scope confirmed

`git diff --stat 7bdecda0a~1 7bdecda0a` matches the report's file list exactly: `main.go`
(+1 line), new `maker/requests.go`, `maker/processor.go`, `socket/handler/maker_skill.go`,
`socket/handler/maker_skill_test.go`, eight seed templates, plus `deploy/shared/routes.conf`
and its generated mirror (outside the brief's stated file list, addressed under finding 2
below). No scope drift beyond that deploy addition, which the implementer flagged itself.

## Focus area 1 — `makerResultFailedValue = 2`

Evidence located, contrary to the implementer's "no source exists" framing:

- `libs/atlas-packet/character/clientbound/maker_result.go`'s `MakerResultFailed` doc and
  `libs/atlas-packet/character/maker_result_body.go`'s `MakerResultFailedBody` doc both
  confirm, from IDA (`wire-derivation.md`), that the client's guard treats **any**
  `nResult` outside `{0, 1}` as the bodyless FAILED arm — there is no single canonical
  wire value, so "the reference client/server value" the implementer was looking for does
  not exist as a fixed constant.
- However, `docs/tasks/task-285-maker-skill-crafting/plan.md:1455-1458` and
  `libs/atlas-packet/character/clientbound/maker_result_test.go:150-153,432`
  (`TestMakerResultFailedByteOutput`, `NewMakerResultFailed(2)`) already established `2` as
  this task family's worked example/precedent for "a value > 1," landed in Task 9's commit.
  `docs/tasks/task-285-maker-skill-crafting/ruling-failed-arm.md:68` cites the same value.
- Task 25's `2` therefore matches existing in-repo precedent, but the implementer's report
  says "I did not find any existing constant... `2` is my own choice" — it did not go
  looking for Task 9's fixture/plan.md, and got lucky rather than grounded. Non-blocking:
  the resulting behavior is correct and consistent with the rest of the codebase, but the
  code comment at `socket/handler/maker_skill.go:17-22` should cite the Task 9 precedent
  (`plan.md:1455`, `maker_result_test.go`) instead of implying it is a fresh, unsourced
  choice — a later reader (Task 26) doing the same "is this invented?" check would waste
  time re-deriving what's already recorded two tasks back.

## Focus area 2 — deploy routing change

Verified independently, not just trusted:

- `deploy/shared/routes.conf`: pre-existing rules for `pets`, `mount`, `quests`, `skills`,
  `macros`, `cash-shop`, `inventory` all use the identical shape
  `location ~ ^/api/characters/[^/]+/<sub>(/.*)?$ { set $u "<svc>:8080"; proxy_pass ... }`.
  The new `maker` rule (line ~146) is byte-for-byte the same shape, placed directly after
  `inventory` and before the generic `^/api/characters(/.*)?$ → atlas-character` catch-all
  at line 445 — confirmed by direct `grep -n`, not assumption. It does not shadow or
  reorder anything: nginx picks the first matching `location ~` in file order, and this
  block is strictly more specific than the trailing catch-all.
- Regenerated `deploy/k8s/base/routes.conf.template.generated` myself with
  `bash tools/gen-routes.sh` and diffed against the committed file: **no drift** — the
  committed generated file already matches what the generator produces from the source
  `routes.conf`, and `ns-vars.generated.yaml` is unaffected (confirms the implementer's
  claim that `NS_ATLAS_MAKER` pre-existed from atlas-maker's own registration).
- The claimed mis-route is real: `services/atlas-maker/atlas.com/maker/main.go`'s
  `/api/` prefix + `craft/resource.go`'s `/characters/{characterId}/maker` sub-router give
  a live path of `/api/characters/{id}/maker/crafts`, which without this rule would fall
  through to the generic `characters` catch-all → `atlas-character`. This is a genuine,
  producible fix within CLAUDE.md's "finish producible work" mandate, not scope creep.

## Focus area 3 — `MAKER` service-URL token

Verified: `grep -rn "INVENTORY" deploy/k8s/base/env-configmap.yaml deploy/k8s/overlays/`
turns up no `INVENTORY_SERVICE_URL` anywhere — only Kafka topic names matching the
substring. `libs/atlas-rest/requests/url.go`'s `RootUrlFor` falls back to `BASE_SERVICE_URL`
plus namespace rewriting when no domain-specific override exists. `INVENTORY`'s
`compartment/requests.go:17-19` and the new `maker/requests.go:22-24` call `RootUrlFor`
identically with only the domain string differing. Since `INVENTORY` has zero
env-configmap/overlay provisioning today, mirroring that (i.e., adding nothing) is the
correct match, not a partial job. No missing token; the brief's "partial job resolves to a
bare string" warning does not apply here because neither service uses a per-domain
override at all.

## Focus area 4 — verbatim forwarding

Read `TestMakerSkillHandlerForwardsEachModeVerbatim`
(`socket/handler/maker_skill_test.go:107-190`) directly. The CREATE fixture
(`makerSkillCreateBytes:28-35`) encodes two gems, `4021313` (commented "not held by the
character") and `4021314`; the assertion at line 124 checks both ids land unchanged in the
captured `maker.CraftRequest.GemItemIds`. This is a real, non-tautological assertion — the
handler has no gem-filtering logic (confirmed by reading `maker_skill.go`), so removing
this test would not silently pass a filtering bug either way, but the fixture and
assertion do prove no filtering occurs today.

## Focus area 5 — FR-5.2 no-lock-up guarantees

All three confirmed with real assertions, not just presence of a call:

- Rejection: `TestMakerSkillHandlerWritesFailureOnRejection` iterates
  `craftErrorTestCases` (12 entries, matching all 12 `Code` constants in
  `services/atlas-maker/atlas.com/maker/craft/errors.go`, confirmed by reading that file)
  and calls `assertMakerResultFailedWritten`, which asserts exactly one announced write,
  the writer name `charcb.MakerResultWriter`, and a 4-byte body (nResult-only, matching
  the bodyless FAILED arm's wire shape).
- Transport/unreachable: `TestMakerSkillHandlerWritesFailureWhenMakerIsUnreachable` closes
  the httptest server before use so any dial fails, then runs the same
  `assertMakerResultFailedWritten` check.
- Acceptance: `TestMakerSkillHandlerWritesNothingOnAcceptance` stubs a 200 response and
  asserts `len(rec.announced) == 0`.

Ran the suite myself: `go test ./socket/handler/... -run MakerSkill -v` — all pass,
including all 12 rejection sub-tests and the unreachable case (verified transport error
message in the captured log: `dial tcp ... connect: connection refused`).

## Focus area 6 — the eight seed templates

- `git diff --stat 7bdecda0a~1 7bdecda0a -- services/atlas-configurations/seed-data/templates/`
  touches exactly the eight named files; `template_gms_12_1.json`, `_48_1`, `_61_1` do not
  appear in the diff.
- Ran `go run ./tools/packet-audit operations --check` from the repo root myself:
  **`operations check OK (0 absent-writer note(s))`, exit 0.**
- Cross-checked every opcode against `docs/packets/registry/<version>.yaml`'s `MAKER_SKILL`
  entries: gms_72=112(0x70), gms_79=111(0x6F), gms_83=113(0x71), gms_84=113(0x71),
  gms_87=116(0x74), gms_92=124(0x7C), gms_95=125(0x7D), jms_185=108(0x6C) — all match the
  committed template values exactly.
- Verified with a Python sweep that each modified template's `socket.handlers` array
  remains strictly ascending by `opCode` after the insertion (72/79/83/84/87/92/95/jms185
  all sorted).
- `options.operations` table (`CREATE:1, CREATE_WITH_UPGRADE:2, MONSTER_CRYSTAL:3,
  DISASSEMBLE:4`) matches `libs/atlas-packet/character/serverbound/maker_skill.go`'s
  `MakerMode*` constants (confirmed by reading that file) on all eight templates.

## Focus area 7 — RED not independently captured

Verified by inspection AND by direct reproduction: created a disposable worktree at
`7bdecda0a~1` (Task 24's head), copied only `maker_skill_test.go` in, and ran
`go test ./socket/handler/... -run MakerSkill -v`. Result:

```
socket/handler/maker_skill_test.go:4:2: package atlas-channel/maker is not in std
FAIL	atlas-channel/socket/handler [setup failed]
```

This confirms the suite is genuinely discriminating — it does not compile, let alone pass,
without the implementation — exactly as the brief's Step 2 predicted (`undefined:
MakerSkillHandleFunc`, or in this case a compile-time package-not-found one step earlier
since the `maker` package doesn't exist yet either). The missing isolated RED transcript is
a process gap, not a test-quality one.

## Other checks

- `main.go`'s `charsb` alias for `libs/atlas-packet/character/serverbound` pre-existed
  (`main.go:101`); the new registration line (`main.go:1021`) uses it correctly, placed in
  the block the brief pointed at.
- `maker/processor.go`'s `CodeOf` defaults to `CodeUnknown` for both a malformed error body
  and a transport failure, matching FR-5.2's requirement that no error path is unhandled.
- Confirmed atlas-maker's `craft.Request` struct (`services/atlas-maker/atlas.com/maker/craft/processor.go:54-69`)
  has no `InventoryType` field, so mode 4's dropped `InventoryType()` is a genuine
  contract-shape fact, not filtering by the channel — consistent with the "forward every
  field the contract has a slot for" claim.
- `go build ./...` and `go test ./socket/handler/... -run MakerSkill -v` in the actual
  worktree: all green, matching the report's captured output.

## Findings

None are blocking. One non-blocking documentation/traceability gap:

- **Non-blocking** — `services/atlas-channel/atlas.com/channel/socket/handler/maker_skill.go:17-22`:
  the `makerResultFailedValue = 2` comment and the task report both frame `2` as an
  unsourced, freshly invented value. It is not: Task 9's byte fixture
  (`libs/atlas-packet/character/clientbound/maker_result_test.go:432`,
  `TestMakerResultFailedByteOutput`) and `docs/tasks/task-285-maker-skill-crafting/plan.md:1455-1458`
  already use `2` as this family's canonical "> 1" example, and
  `docs/tasks/task-285-maker-skill-crafting/ruling-failed-arm.md:68` records it. The value
  chosen is correct and consistent with precedent, but the comment should cite that
  precedent rather than presenting `2` as a bare, ungrounded choice — otherwise Task 26
  (which reads this same constant) has to redo this exact archaeology.

## Not evaluable

None. All seven focus areas were directly verifiable within this commit's diff plus the
files it calls into (packet library, atlas-maker's craft contract, deploy routing config,
and the task's own recorded rulings/plan).
