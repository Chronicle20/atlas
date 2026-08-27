# task-146 — planning context

Companion to `plan.md`. Records the key files, the decisions the plan bakes in,
the dependency order, and what an implementer must not re-derive.

## Where the work lives

| Area | Path | Role |
|---|---|---|
| Registry | `docs/packets/registry/gms_v95.yaml` | op → opcode / fname / packet link. Tasks 1–4. |
| Templates | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` | `.socket.handlers` (154) + `.socket.writers` (239). Tasks 6–11. |
| Dispatcher docs | `docs/packets/dispatchers/*.yaml` | source of truth for `options.operations`. Six new handler-keyed docs, Tasks 7–8. |
| Codecs | `libs/atlas-packet/**` | fixtures + markers. Tasks 11–15. |
| Evidence | `docs/packets/evidence/gms_v95/*.yaml` | 702 records today; 4 new. |
| Grading | `tools/packet-audit/internal/matrix/{build,grade}.go` | how a cell promotes. Task 5 changes `build.go`. |
| Matrix | `docs/packets/audits/{STATUS.md,status.json}` | regenerated, never hand-edited. |

Module roots (the `go build` / `go test` cwd): `libs/atlas-packet`,
`tools/packet-audit`, `services/atlas-channel/atlas.com/channel`,
`services/atlas-login/atlas.com/login`.

## Dependency order

```
Task 1  registry opcode fix (AUTO_DISTRIBUTE_AP 98 -> 99)
Task 2  fname promotions x4          -> flips 4 cells to verified immediately
Task 3  packet: links x4             -> flips STORAGE; arms 3 more
Task 4  LOGIN_AUTH seed resolution
Task 5  packet-audit allowlist       -> flips NpcAskSlideMenu sub-struct
        |
Task 6  template routing x11         <- needs Task 1's opcode
Task 7  dispatcher docs (party/guild/buddy)
Task 8  dispatcher docs (messenger/BBS/storage)  <- generates the 12th W1 entry
Task 9  messageType
Task 10 failedReasonCodes
Task 11 movement types + fixture coupling
        |
Task 12 CHECK_SPW_RESULT   \
Task 13 MULTI_CHAT          }  <- need Task 3's packet: links and Task 6's routes
Task 14 ENTER_CASHSHOP     /
Task 15 NPC_ACTION            (independent — already linked)
Task 16 n-a proofs
Task 17 coverage manifest + full gate sweep
```

Tasks 1–5 are independent of each other and can run in any order; each
regenerates the matrix, so serialize the commits to keep `status.json` diffs
readable. Tasks 12–14 must follow Tasks 3 and 6 — a serverbound cell needs its
op routed in the template (playbook §9) and its `packet:` link present.

## Decisions baked into the plan

**Design corrections (plan §0).** Seven claims in `design.md` are wrong or stale
and the plan overrides them: the AP opcode blocker is resolved (99, not a shared
98); three of the "tier-0" rows are actually tier-1; `CLogin::OnCheckSPWResult`
is in the IDB and the packet is one byte, not bodyless; `chat/serverbound/Multi`
already carries its v95 gate; `NpcAskSlideMenuConversationDetail` needs a
tooling allowlist entry, not a fixture; the templates omit `options` rather than
setting it `null`; and two CI gates were missing from the design's list.

**Open Question 1 — family scope.** No `dispatcher-family-implementer` pass.
`docs/packets/evidence/families.yaml`'s `dispatchers:` list is empty, so nothing
in scope is capped at 🧩, and `dispatcher-lint` reports clean. Arm coverage is
still completed because it is cheap: storage's four arms are already verified,
and the one apparent NPC_TALK cb gap turned out to be a grading artifact
(Task 5).

**Open Question 3 — inert routes.** Resolved clean. All twelve W1 symbols have
live registrations; the plan tables the exact `main.go` line for each so the
implementer does not re-grep.

**Open Question 4 — CHANGE_MAP alts.** `CField::SendTransferFieldRequest` is the
one export-resident, report-backed sender; the two `CCashShop`/`CITC` names are
CSV aliases and stay as `fname_alts`. If a genuinely distinct transfer build
site turns up, it is a second codec and a second cell — surfaced, not folded.

**Open Question 5 — n-a mechanism.** No new matrix rows, and no
`feature-na-evidence.yaml` entries: that file is only consulted when an `n-a`
cell has a `verified` same-feature sibling, and none of the three W4 symbols
belongs to a family in `feature-families.yaml`. Proofs live in the task folder;
`ServerLoad` alone may earn an `_unimplemented.json` entry.

**PRD deviations to expect.** PRD §7 said `tools/packet-audit` needs no change —
Task 5 changes it. PRD §4.3 described the gap as `"options": null` — it is a
missing key. PRD §4.1 requires routing `PartyInviteRejectHandle`; Task 6 Step 3
makes that conditional on an IDB confirmation, because `DENY_PARTY_REQUEST` is
`n-a` on v95 and the v95 reject rides `PARTY_RESULT` sb 146 instead.

## Deliberately large task (F4 advisory)

**Task 11 lists 7 files** against Step 5a's ~6 threshold, and is kept whole on
purpose. It is a single derivation — one movement-attribute table read out of
the v95 client — written into four template writer entries (all in **one** file)
and then pinned by three test files plus one new shared helper. Splitting it
would put the template write and the fixture coupling in different tasks, which
is precisely the desync the coupling exists to prevent: the array and the
assertion that guards it must land in one commit. The file count is wide, the
change is narrow, and no other task in the plan touches those files.

Everything else is at or under the threshold, and no task spans more than one
service.

## What not to re-derive

The plan's §1 table pins fourteen v95 IDB addresses resolved during planning,
plus the two that are still unresolved and how to find them. The IDB uses
**MSVC-mangled** names (`?OnCheckSPWResult@CLogin@@IAEXAAVCInPacket@@@Z`), so a
plain `CClass::Method` lookup returns "Not found" — that is a naming artifact,
not an absent function. Session id for the v95 IDB is `ecc757f4`
(`GMS_v95.0_U_DEVM.exe.i64`) on `http://192.168.20.3:8745/mcp`.

All seven CI gates were verified exit 0 at the branch point; the two standing
`matrix --check` notes (`CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79`,
`USE_TELEPORT_ROCK × gms_v48`) are expected and not findings.

## As-built

Written by Task 17 after all 17 plan tasks landed, from `progress.md`, the
per-task reports, and a direct `git diff 43975545a..HEAD` sweep (the branch's
starting point before Task 1). Full derivation lives in
`docs/tasks/task-146-v95-packet-verification-batch/coverage-manifest.yaml`.

### Execution order

Actual order was 6, 2, 3, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17 — Tasks
1–5 (independent, any order) were resequenced by the controller so Task 6's
template routes existed before the later verification tasks needed them, and
Task 1 (the AP opcode fix) landed as part of the same commit range as Task 6.
All landed commits are `11cfdd31b..ac0a624c6` on top of base `43975545a`.

### `PartyInviteRejectHandle` — routed, not dropped

Design flagged this as conditional (PRD §4.1 required routing it; the plan
made it conditional on an IDB confirmation because `DENY_PARTY_REQUEST` is
`n-a` on v95). Task 6 confirmed it IS live: `CWvsContext::OnPartyResult`
(`0xa10ab0`) builds `COutPacket(&oPacket, 146)` on the decline path, so
`PartyInviteRejectHandle` was routed at opCode `0x92`, IDB-confirmed. No
deviation occurred — the conditional resolved to "route it."

### Four full coverage-matrix verifications (Tasks 12–15)

All on `gms_v95`, all `incomplete -> verified`:

- `CHECK_SPW_RESULT` clientbound (`login/clientbound/LoginPicResult`), Task 12, 30eaffea3
- `MULTI_CHAT` serverbound (`chat/serverbound/ChatMulti`), Task 13, 5895eb521
- `ENTER_CASHSHOP` serverbound (`cash/serverbound/CashShopEntry`), Task 14, 06205b96d
- `NPC_ACTION` clientbound (`npc/clientbound/NpcAction`), Task 15, aa038b1e7

Plus four more promoted by Tasks 2/3/6 (fname promotion + template routing,
no new fixture needed because the codec already existed): `CHANGE_MAP` sb,
`NPC_TALK` sb, `NPC_TALK` cb, `CHANGE_MAP_SPECIAL` sb, `STORAGE` sb (five,
not four — see coverage-manifest.yaml for the full per-op reasoning). `
LOGIN_AUTH` cb resolved to `n-a` (Task 4, genuinely absent — proof in
`login-auth-resolution.md`). `AUTO_DISTRIBUTE_AP` sb had its opcode corrected
98 -> 99 without a state change (Task 1). The sub-struct
`npc/clientbound/NpcAskSlideMenuConversationDetail` was promoted by a
grading-allowlist fix, not new codec work (Task 5).

### Fixed regression: four sub-struct rows demoted `verified -> incomplete`

Side effect of Task 2's fname promotions, first surfaced and deferred by the
Task 2 reviewer, confirmed by direct `status.json` diff, and fixed in a
post-plan fix round. Sub-struct rows are NOT graded through `findReport`
(that path, `grade.go:282-292`, serves op rows only); they are graded through
`Build`'s `usedWriters`/`protectedWriters` logic (the sub-struct skip/protect
gate, `build.go:363-395`, and `worstCandidateCell`'s `used[vk][wn]` /
`protected[vk][wn]` assignment, `build.go:514-516`). Each promoted op fname now equals `baseFName(IDAName)` of one of
these four sub-structs' own reports, so the op row's `worstCandidateCell`
consumed that sub-struct's writer as a used candidate, and the sub-struct was
skipped (gap-filled to `incomplete / "no audit report"`) instead of grading
from its own evidence — even though nothing about the underlying codec
changed. Confirmed regressed on `gms_v95`, all `verified -> incomplete`:

- `field/serverbound/FieldChange` (sibling of the promoted `CHANGE_MAP` sb)
- `npc/clientbound/NpcNpcConversation` (sibling of the promoted `NPC_TALK` cb)
- `npc/serverbound/NpcStartConversation` (sibling of the promoted `NPC_TALK` sb)
- `portal/serverbound/PortalScript` (sibling of the promoted `CHANGE_MAP_SPECIAL` sb)

Fixed by adding four version-scoped `legacyConsumedSiblingWriters` entries in
`tools/packet-audit/internal/matrix/build.go`, following the precedent Task 5
established for `NPC_TALK_MORE`/`NpcAskSlideMenuConversationDetail`: each
entry is keyed by `versionScopedOpKey(op, dir, "gms_v95")` so only the gms_v95
collision is un-suppressed, and each sub-struct escapes consumption only when
its own pinned evidence independently grades `verified` — not a
force-promote. `status.json` diff confirms exactly these four cells moved
`incomplete -> verified` on `gms_v95` and no other cell changed on any
version. `matrix --check` is clean at the fixed HEAD.

### Open item for final pre-PR triage: T13's spliced IDA-export guard

`docs/packets/ida-exports/gms_v95.json`'s spliced entry for
`CUIStatusBar::SendGroupMessage` (added by Task 13) carries a `calls` array
whose 4 elements all bear an identical, unexplained guard
`nChatTarget == 4 && nMemberCnt`, and omits the recipient loop's unguarded
`Encode4`. The Task 13 reviewer traced this to the harvester/parser, not an
implementer fabrication, and confirmed by grep that no gate reads `idasrc`
export data for grading — inert for Task 13's verified claim, but it is
committed shared ground truth a future chat-domain task could be misled by.
Task 14's own splice (`CWvsContext::SendMigrateToShopRequest`) was checked
for the same defect by its reviewer and is clean — `calls: null`.

### Two older deferred doc-completeness minors (non-blocking, documentation only)

- **Task 2**: the first implementation attempt's report predicted the four
  promoted ops' sibling sub-struct rows would "re-subsume cleanly into
  verified"; they instead landed `incomplete / no audit report` at the time
  — correct per the `usedWriters`/`protectedWriters` op-row consumption path
  in `build.go`, not the prediction, though the prediction was never
  reconciled in the report. (This is the same underlying mechanism as the
  "fixed regression" section above — recorded twice here because the Task 2
  review flagged it as a doc-completeness gap independently of the later
  mechanical status.json diff that confirmed it; the four cells are now
  fixed back to `verified` by the post-plan fix round.)
- **Task 4**: `login-auth-resolution.md`'s "mandatory sibling cross-check"
  searched only `func_query filter=*SPW*` and never cited the send-side
  siblings already present in the same registry file
  (`docs/packets/registry/gms_v95.yaml:2311` `REGISTER_PIC`, `:2316`
  `CHAR_SELECT_WITH_PIC`, `:2326` `VIEW_ALL_WITH_PIC`). Outcome-neutral — the
  reviewer located them independently and the primary evidence (the
  exhaustive, address-cited `CLogin::OnPacket` switch enumeration) already
  rules out a missed `LOGIN_AUTH` arm regardless.

### `plan.md` carries a known-wrong derivation — deliberately left uncorrected

`docs/tasks/task-146-v95-packet-verification-batch/plan.md` lines ~1728 and
~1734 assert `qualifiedWriterName("login", "ServerLoad") = LoginServerLoad`.
This is WRONG: `run.go`'s login candidates always carry an empty `pkg`
(`pkg: "login"` appears in none of the 27 enumerated pkg literals), so
`qualifiedWriterName("", "ServerLoad")` short-circuits on `pkg == ""` and
returns the bare name `ServerLoad` — confirmed by the sibling gms_v95 reports
for other bare-pkg login candidates (`ServerStatus.json`, `AfterLogin.json`,
`ServerIP.json`), which all carry a `WriterName` equal to the unprefixed
struct name. This wrong derivation caused a Task 16 blocking review finding
(`_unimplemented.json` originally recorded `login/clientbound/LoginServerLoad`,
fixed to `login/clientbound/ServerLoad` in commit `ac0a624c6`). `plan.md` is a
historical phase artifact and is deliberately NOT being rewritten to correct
it — recorded here so a future reader of `plan.md` is not misled by those two
lines.

### Standing pattern: a brief can be wrong about mechanics, not just values

Four instances on this branch, each a brief asserting a derivation the
implementer then propagated without re-running it:

1. **Task 12** — the brief treated evidence `category` as an open field; it
   is a closed enum, caught before commit.
2. **Task 13** — the brief's cited line numbers in `multi.go` were stale
   after earlier tasks landed; the implementer located the site by content.
3. **Task 14** — the brief's Step 5 `git add` list omitted
   `docs/packets/ida-exports/gms_v95.json`, though the brief's own `### Files`
   section required it; the implementer followed the Files inventory instead.
4. **Task 16** — the brief supplied the wrong `qualifiedWriterName` derivation
   above; the implementer read and propagated it rather than re-executing
   `run.go`'s actual `pkg == "" -> return name` short-circuit. This is the
   only one of the four that reached a blocking review finding, precisely
   because the other three were caught by re-deriving rather than trusting
   the brief's stated mechanics.

Forward-looking lesson for future plans in this repo: a brief-supplied
DERIVATION (a qualified name, a computed opcode, a closed-enum check) must be
re-executed against the actual tool/source at implementation time, not
accepted on the brief's authority — the brief is a lead, not ground truth,
exactly as CLAUDE.md's evidence-and-grounding section already states for
remembered facts generally.

### Gate sweep (Step 2)

Each run standalone from the worktree root unless noted:

- `cd tools/packet-audit && go test ./...` — `FAIL` in package `cmd` only:
  `TestSeedFName_RealTemplatesInsertionCoverage`, the standing known-unrelated
  failure (confirmed unrelated to this branch's diff by the Task 5 reviewer's
  reverse-apply check). All other `tools/packet-audit` packages pass.
- `go run ./tools/packet-audit fname-doc --check` — exit 0
  ("271 structs without an audit report carry no fname").
- `go run ./tools/packet-audit operations --check` — exit 0
  ("0 absent-writer note(s)").
- `go run ./tools/packet-audit dispatcher-lint` — exit 0 ("clean").
- `go run ./tools/packet-audit doc-freshness --check` — exit 0.
- `go run ./tools/packet-audit gate-check --check` — exit 0
  ("21 gate(s) have verified byte-fixtures ... 1 partial-by-design" — the new
  Task 13 `ChatMulti` v92/v95 boundary gate).
- `go run ./tools/packet-audit matrix --check` — exit 0, only the two
  standing expected notes (`CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79`,
  `USE_TELEPORT_ROCK × gms_v48`).
- `tools/verify.sh` (flagless) — see Task 17's report for the exit code.

### Acceptance criteria (Step 3) — confirmed mechanically against `status.json`

- All nine PRD §4.2 cells read `verified` on `gms_v95`: `ENTER_CASHSHOP` sb,
  `MULTI_CHAT` sb, `CHANGE_MAP` sb, `NPC_TALK` sb, `CHANGE_MAP_SPECIAL` sb,
  `STORAGE` sb, `NPC_TALK` cb, `CHECK_SPW_RESULT` cb, `NPC_ACTION` cb.
- None of the PRD §4.4 cells changed state (17 cells checked directly against
  the pre-branch `status.json`, all identical).
- No non-v95 template file appears in `git diff --name-only
  43975545a..HEAD` — the only template changed is
  `template_gms_95_1.json`.
- All twelve PRD §4.3 entries carry a populated `options` block in the
  current template.
- W1: `CharacterAutoDistributeApHandle` routes at `0x63` (not `0x62`),
  confirmed directly in the current template.

No deviation found beyond the two already discussed above
(`PartyInviteRejectHandle` routed as expected; the four sub-struct
regressions, which are a side effect of Task 2, not a Step 3 acceptance-check
failure).
