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

_To be filled in by Task 17 with the actual outcome, including any deviation
from the plan — notably whether `PartyInviteRejectHandle` was routed._
