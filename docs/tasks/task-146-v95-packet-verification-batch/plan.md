# GMS v95.1 Packet Verification & Template Completeness — Implementation Plan

Task: `task-146-v95-packet-verification-batch`
Inputs: `docs/tasks/task-146-v95-packet-verification-batch/prd.md`,
`docs/tasks/task-146-v95-packet-verification-batch/design.md`
Branch: `task-146-v95-packet-verification-batch` · base `92cb7e4dd`

---

## 0. Corrections to design.md established during planning

`design.md` was written before a live v95 IDB read and before a full
`status.json` tier/state sweep. Seven of its claims are wrong or stale. Each
correction below is evidence-backed and is what the tasks implement — where a
task contradicts `design.md`, **this plan wins**.

**C1 — Blocker B1 is RESOLVED, and the registry is wrong, not ambiguous.**
Live decompile of the GMS v95.0_U_DEVM IDB (session `ecc757f4`):

```
CWvsContext::SendAbilityUpRequest#DistributeAp      @0x9f61c0
  COutPacket::COutPacket(&oPacket, 98);  /*0x9f6330*/
  Encode4(update_time); Encode4(dwFlag); SendPacket();

CWvsContext::SendAbilityUpRequest#AutoDistributeAp  @0x9f63b0
  COutPacket::COutPacket(&oPacket, 99);  /*0x9f6535*/
  Encode4(update_time); Encode4(nValue);
  loop { Encode4(dwStatFlag); Encode4(nValue); }  SendPacket();
```

So `DISTRIBUTE_AP` sb = **98 (0x62)** (registry correct) and
`AUTO_DISTRIBUTE_AP` sb = **99 (0x63)** (registry says 98 — wrong, despite its
`provenance: manual` / "IDB-verified task-085" note at
`docs/packets/registry/gms_v95.yaml:2736-2741`). This mirrors v83, where
`template_gms_83_1.json` routes `CharacterDistributeApHandle` at `0x57` (87) and
`CharacterAutoDistributeApHandle` at `0x58` (88) — adjacent, not shared.
Consequence: W1 keeps all 12 entries; `CharacterAutoDistributeApHandle` routes
at **`0x63`** (free in the v95 template); nothing is displaced.

**C2 — the tier table in design §2.3 is wrong.** From `status.json`
(`tier1` per op row): `MULTI_CHAT` sb is **tier-1**, `CHANGE_MAP` sb tier-1,
`NPC_TALK` sb tier-1, `NPC_ACTION` cb tier-1, `PARTY_RESULT` sb tier-1,
`DISTRIBUTE_AP`/`AUTO_DISTRIBUTE_AP` tier-1. Tier-**0**: `ENTER_CASHSHOP` sb,
`CHECK_SPW_RESULT` cb, `STORAGE` sb, `CHANGE_MAP_SPECIAL` sb, `NPC_TALK` cb.
So the design's "three tier-0 evidence pins against playbook §7" is **two**
(`ENTER_CASHSHOP`, `CHECK_SPW_RESULT`); `MULTI_CHAT` pins evidence as an
ordinary tier-1 cell, no exception needed.

**C3 — `CLogin::OnCheckSPWResult` is NOT absent from the IDB, and the packet is
not bodyless.** It resolves at **`0x5d23f0`** (mangled
`?OnCheckSPWResult@CLogin@@IAEXAAVCInPacket@@@Z`) and decompiles to:

```c
void __thiscall CLogin::OnCheckSPWResult(CLogin *this, CInPacket *iPacket)
{
  CInPacket::Decode1(iPacket);              /*0x5d23f7*/
  CLoginUtilDlg::Error(93, &this->m_pChildModal); /*0x5d2405*/
  this->m_bRequestSent = 0;                 /*0x5d240d*/
}
```

One byte. `libs/atlas-packet/login/clientbound/pic_result.go:19-25` already
writes exactly `w.WriteByte(0)`. **No wire divergence; the golden fixture is
`[]byte{0x00}`.** (It is absent from the committed export
`docs/packets/ida-exports/gms_v95.json`, which is why no report exists — that
part of the design holds, and the `packet:` route is still correct.)

**C4 — `chat/serverbound/Multi` already implements the v95 gate.**
`libs/atlas-packet/chat/serverbound/multi.go:53-56` and `:71-77`:

```go
t := tenant.MustFromContext(ctx)
hasUpdateTime := t.Region() == "GMS" && t.MajorVersion() >= 95
```

`run.go:1775-1783`'s comment ("the updateTime prefix is additional in v95")
predates that gate. Design §5 row 2's implied wire fix is **not needed**; the
work is the golden fixture + evidence + a `gates.yaml` row.

**C5 — `NpcAskSlideMenuConversationDetail` is not a missing fixture.** It
already has all three artifacts on v95:

- marker `libs/atlas-packet/npc/clientbound/conversation_test.go:25`
  (`version=gms_v95 ida=0x6dbe50`)
- evidence `docs/packets/evidence/gms_v95/npc.clientbound.NpcAskSlideMenuConversationDetail.yaml`
- report `docs/packets/audits/gms_v95/NpcAskSlideMenuConversationDetail.json`
  (`IDAName: CScriptMan::OnAskSlideMenu#AskSlideMenu`)

Its `incomplete / no audit report` cell is a **report-consumption artifact**:
`docs/packets/registry/gms_v95.yaml:2519-2522` gives `NPC_TALK_MORE` serverbound
the primary `fname: CScriptMan::OnAskSlideMenu`, and
`tools/packet-audit/internal/matrix/build.go:122-134` indexes reports by
`baseFName(r.IDAName)` — which for this writer is exactly
`CScriptMan::OnAskSlideMenu`. The op row therefore consumes the writer
(`usedWriters`), the sub-struct pass skips it (`build.go:255-270`), and the
gap-fill branch (`build.go:288-300`) stamps `no audit report`. Sibling
`NpcSayImageConversationDetail` escapes because `CScriptMan::OnSayImage` is only
an `fname_alt`. The fix is the existing `legacyConsumedSiblingWriters` allowlist
(`build.go:113-118`) — a `tools/packet-audit` change, which PRD §7 said would
not be needed. It is needed, it is small, and it is the documented mechanism
(NOTE_ACTION and USE_CASH_ITEM precedents in the same file).

**C6 — the templates have no `"options": null`.** `grep -c '"options": null'
services/atlas-configurations/seed-data/templates/template_gms_95_1.json` = 0.
The twelve W3 entries simply **omit** the `options` key. Also the JSON path is
**`.socket.handlers` / `.socket.writers`** — there is no `data.attributes`
envelope. File is LF-only (9895 LF lines, 0 CRLF); both arrays are sorted
ascending by numeric `opCode`.

**C7 — two gates the design omitted.** `.github/workflows/packet-matrix.yml`
runs seven blocking gates, not four. `doc-freshness --check` and
`gate-check --check` are also blocking. All seven verified green at branch point
this session (see §1).

---

## 1. Global constraints

**Worktree.** Every command runs from `<repo-root>/.worktrees/task-146-v95-packet-verification-batch`.
Never edit `docs/tasks/` under the main checkout.

**Module roots** (the `go build` / `go test` cwd):

| Module root | Import path |
|---|---|
| `libs/atlas-packet` | `github.com/Chronicle20/atlas/libs/atlas-packet` |
| `tools/packet-audit` | `github.com/Chronicle20/atlas/tools/packet-audit` |
| `services/atlas-channel/atlas.com/channel` | `atlas-channel` |
| `services/atlas-login/atlas.com/login` | `atlas-login` |

**Gate baseline — all seven exit 0 on the branch point.** Verified this
session; run each as its own literal command line (a shell loop over a variable
mangles the args here):

```
cd tools/packet-audit && go test ./...
go run ./tools/packet-audit fname-doc --check      # OK (271 structs w/o report carry no fname)
go run ./tools/packet-audit operations --check     # OK (0 absent-writer notes)
go run ./tools/packet-audit dispatcher-lint        # dispatcher-lint: clean
go run ./tools/packet-audit doc-freshness --check  # 10 versions, 7 CI gates
go run ./tools/packet-audit gate-check --check     # 20 gates, both sides verified
go run ./tools/packet-audit matrix --check         # exit 0, 2 expected n-a-evidence notes
```

`matrix --check`'s two standing notes are
`CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79` and `USE_TELEPORT_ROCK × gms_v48`
— expected, not findings. Regenerate with `go run ./tools/packet-audit matrix`
(no flags) and commit `STATUS.md` + `status.json` in the same commit as the
change that moved them.

**IDA access.** The ida-pro MCP server at `http://192.168.20.3:8745/mcp` is
live. `idb_list` shows ten adopted sessions; the v95 one is
**`ecc757f4` = `GMS_v95.0_U_DEVM.exe.i64`** — pass it as the `database`
argument on every call. **The IDB uses MSVC-mangled names**: `lookup_funcs`
with `CLogin::OnCheckSPWResult` returns "Not found"; with
`?OnCheckSPWResult@CLogin@@IAEXAAVCInPacket@@@Z` it returns `0x5d23f0`. Use
`func_query`/`find_bytes` by pattern when the mangling is unknown, and per
playbook §10 **distrust the symbol** — ground truth is the integer in
`COutPacket::COutPacket(&pkt, N)`.

**Addresses resolved during planning** (do not re-hunt these):

| Function | v95 address | In committed export? |
|---|---|---|
| `CWvsContext::SendAbilityUpRequest#DistributeAp` | `0x9f61c0` | yes |
| `CWvsContext::SendAbilityUpRequest#AutoDistributeAp` | `0x9f63b0` | yes |
| `CField::SendTransferFieldRequest` | `0x5345c0` | yes |
| `CUserLocal::TalkToNpc` | `0x9321f0` | yes |
| `CUserLocal::CheckPortal_Collision` | `0x919a10` | yes |
| `CScriptMan::OnScriptMessage` | `0x6de0f0` | yes |
| `CScriptMan::OnAskSlideMenu#AskSlideMenu` | `0x6dbe50` | yes |
| `CNpc::OnMove` | `0x678060` | yes |
| `CTrunkDlg::OnPacket#Operation` | `0x76a990` | yes |
| `CUserRemote::OnMove` | `0x948a80` | yes |
| `CMob::OnMove` | `0x6521e0` | yes |
| `CPet::OnMove` | `0x69fb60` | yes |
| `CLogin::OnCheckPasswordResult` | `0x5dc600` | no |
| `CLogin::OnCheckSPWResult` | `0x5d23f0` | no |
| `CWvsContext::SendMigrateToShopRequest` | **unresolved** | no |
| `CUIStatusBar::SendGroupMessage` | **unresolved** | no |

The two unresolved names failed a mangled-name guess; find them by the playbook
§10 byte signature `6A <op> 8D 8D ?? ?? ?? ?? E8` (opcode 43 = `0x2B`,
140 = `0x8C`) or by `find_bytes`/`insn_query` over the `COutPacket` ctor xrefs.
**An unnamed sender is not an absent sender** — name it, do not defer.

**Never do.** Never copy a mode value, type code, reason code or opcode from
another version's template. Never overwrite `docs/packets/ida-exports/gms_v95.json`
(playbook §10 — the export is not idempotent). Never modify a non-v95 template.
Never change an already-verified version's wire encoding; if a divergence is
proven, it lands as its own commit with its own review.

**Test fixtures.** `libs/atlas-packet/test` supplies `pt.Variants`
(`test/context.go:18-42`; **GMS v95 is `pt.Variants[3]`**), `pt.CreateContext`,
`pt.Encode`, `pt.RoundTrip`. Copy the byte-golden shape rather than inventing
one.

**Marker format** (playbook §6), one line immediately above the test func:

```
// packet-audit:verify packet=<pkg>/<dir>/<Qualified> version=gms_v95 ida=<0xaddr>
```

`<Qualified>` = `qualifiedWriterName(pkg, name)` = TitleCase(pkg)+structName
(`tools/packet-audit/cmd/run.go:223-228`). Confirmed ids for this task:
`cash` + `ShopEntry` → `CashShopEntry`; `chat` + `Multi` → `ChatMulti`;
`login` + `PicResult` → `LoginPicResult`.

**Evidence record schema** (playbook §7; written by `evidence pin`, then hand-add
`verifies:`). Model —
`docs/packets/evidence/gms_v95/buddy.serverbound.BuddyOperationAccept.yaml`:

```yaml
packet: buddy/serverbound/BuddyOperationAccept
direction: serverbound
version: gms_v95
category: TIER1-FIXTURE
ida:
    function: CField::SendAcceptFriendMsg
    address: "0x52f290"
    decompile_sha256: 1980c576...
```

Filename convention: `<pkg>.<direction>.<Qualified>.yaml`.

**Commit discipline.** One commit per task. Registry/template/tooling edits that
move a cell commit `STATUS.md` + `status.json` alongside. Never commit to `main`.

---

## Task 1: W0a — correct the `AUTO_DISTRIBUTE_AP` v95 opcode (98 → 99)

The evidence is in §0/C1 and needs no re-derivation. Re-confirming it live is
cheap and allowed; inventing anything else is not.

### Files

- `docs/packets/registry/gms_v95.yaml` — entry at line 2736 (`AUTO_DISTRIBUTE_AP`, `direction: serverbound`); change `opcode: 98` → `opcode: 99`, add an `ida:` block and replace the stale `note:`
- `docs/packets/audits/STATUS.md` — regenerated, do not hand-edit
- `docs/packets/audits/status.json` — regenerated, do not hand-edit

Patterns to copy: `docs/packets/registry/gms_v95.yaml:225-239` (`CLAIM_RESULT` — the only entry shape in this file that carries both `ida:` and a block `note:`)

- [ ] **Step 1: Re-confirm the two opcodes against the live IDB**

Decompile `0x9f61c0` and `0x9f63b0` on database `ecc757f4` and record the
`COutPacket::COutPacket(&oPacket, N)` integer for each. Expected, from planning:
`0x9f61c0` → **98** at build site `0x9f6330`; `0x9f63b0` → **99** at build site
`0x9f6535`. If either differs, STOP and report — do not edit the registry.

- [ ] **Step 2: Edit the registry entry**

Replace the `AUTO_DISTRIBUTE_AP` serverbound entry (line 2736ff) with:

```yaml
- op: AUTO_DISTRIBUTE_AP
  direction: serverbound
  opcode: 99
  fname: CWvsContext::SendAbilityUpRequest
  provenance: manual
  ida:
    address: 0x9f63b0
  note: >-
    task-146: opcode corrected 98 -> 99. The prior manual entry (task-085)
    claimed 98, which is DISTRIBUTE_AP's opcode. Live GMS v95.0_U_DEVM read:
    SendAbilityUpRequest#AutoDistributeAp @0x9f63b0 builds
    COutPacket(&oPacket, 99) @0x9f6535 (Encode4 update_time, Encode4 nValue,
    then a loop of Encode4 dwStatFlag / Encode4 nValue pairs);
    #DistributeAp @0x9f61c0 builds COutPacket(&oPacket, 98) @0x9f6330
    (Encode4 update_time, Encode4 dwFlag). Matches the v83 split
    (0x57 / 0x58).
```

Leave the `DISTRIBUTE_AP` entry (line 2731) untouched.

- [ ] **Step 3: Regenerate and gate**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected: exit 0. `AUTO_DISTRIBUTE_AP` sb `gms_v95` must remain `verified` (it
grades off the shared `CWvsContext::SendAbilityUpRequest` report, not the
opcode) and its `opcode` field in `status.json` must now read `99`.
`DISTRIBUTE_AP` sb `gms_v95` must remain `verified` at `98`. Any other cell
changing state is a finding — report it, do not absorb it.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/registry/gms_v95.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "fix(packets): correct gms_v95 AUTO_DISTRIBUTE_AP opcode to 99 (IDB-derived)"
```

---

## Task 2: W0b — promote four v95 `fname`s to their export-resident senders

Four ops carry a `csv-import` primary `fname` that does not exist in the
committed v95 export, while a proven alt does. `grade.go:282-292` (`findReport`)
joins the op to its report by **primary `fname` only**, so the cell cannot see
the report it already has. Swap primary ↔ alt; never delete the CSV name.

### Files

- `docs/packets/registry/gms_v95.yaml` — entries at lines 2378 (`CHANGE_MAP` sb), 2507 (`NPC_TALK` sb), 2818 (`CHANGE_MAP_SPECIAL` sb), 1895 (`NPC_TALK` cb)
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated
- `docs/tasks/task-146-v95-packet-verification-batch/fname-promotions.md` — **new file**; the per-promotion evidence record required by Step 1

Read-only references (the reports these promotions unlock — do not edit):
`docs/packets/audits/gms_v95/FieldChange.json`,
`docs/packets/audits/gms_v95/NpcStartConversation.json`,
`docs/packets/audits/gms_v95/PortalScript.json`,
`docs/packets/audits/gms_v95/NpcNpcConversation.json`

- [ ] **Step 1: Confirm each promoted function live, then write the evidence file**

For each of the four addresses below, decompile on database `ecc757f4` and
record (a) the `COutPacket::COutPacket(&pkt, N)` opcode for a send, or the
dispatch arm for a receive, and (b) that it matches the registry opcode.
`tools/packet-audit/cmd/run.go` already documents these, but the comments are
derived — re-check them live.

| Op × dir | Promote to | Address | Registry opcode must match |
|---|---|---|---|
| `CHANGE_MAP` sb | `CField::SendTransferFieldRequest` | `0x5345c0` | 41 (`0x29`) |
| `NPC_TALK` sb | `CUserLocal::TalkToNpc` | `0x9321f0` | 63 (`0x3F`) |
| `CHANGE_MAP_SPECIAL` sb | `CUserLocal::CheckPortal_Collision` | `0x919a10` | 112 (`0x70`) |
| `NPC_TALK` cb | `CScriptMan::OnScriptMessage` | `0x6de0f0` | 363 (`0x16B`) — receive, so confirm the `nType == 363` dispatch arm reached from `CScriptMan::OnPacket` |

Write each finding into `fname-promotions.md` as: op, promoted fname, address,
build-site or dispatch-arm address, opcode read, verdict. A promotion whose
opcode does not match is **not admissible** — stop and report it as a possible
second codec (design §5.2), do not fold it in.

- [ ] **Step 2: Apply the four swaps**

Each edit is the same shape — promote the alt, demote the CSV name into
`fname_alts` (append, keep any existing alts):

```yaml
# line 2378 — before
- op: CHANGE_MAP
  direction: serverbound
  opcode: 41
  fname: CCashShop::SendTransferFieldPacket
  fname_alts:
    - CField::SendTransferFieldRequest
    - CITC::SendTransferFieldPacket
  provenance: csv-import

# after
- op: CHANGE_MAP
  direction: serverbound
  opcode: 41
  fname: CField::SendTransferFieldRequest
  fname_alts:
    - CCashShop::SendTransferFieldPacket
    - CITC::SendTransferFieldPacket
  provenance: manual
  note: >-
    task-146: primary fname promoted from the csv-import name
    CCashShop::SendTransferFieldPacket (absent from
    docs/packets/ida-exports/gms_v95.json) to the export-resident,
    report-backed sender CField::SendTransferFieldRequest@0x5345c0.
    findReport joins by primary fname only (grade.go:282-292). The CSV
    names are retained as alts. Evidence: fname-promotions.md.
```

`NPC_TALK` sb: `CNpc::ShowQuestList` → `CUserLocal::TalkToNpc`.
`CHANGE_MAP_SPECIAL` sb: `CUserLocal::HandleUpKeyDown` → `CUserLocal::CheckPortal_Collision`.
`NPC_TALK` cb: `CScriptMan::OnPacket` → `CScriptMan::OnScriptMessage` (this
entry has no `fname_alts` today — create the list with the demoted name).

`STORAGE` sb is deliberately **not** promoted here: its four Atlas codecs are
four distinct senders and promoting one would make the op row grade that arm
alone. It takes the `packet:` route in Task 3.

- [ ] **Step 3: Regenerate and check the four cells**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected `gms_v95` transitions in `status.json` (op rows), all from
`incomplete / no audit report`:

| Op × dir | Expected new state | Why |
|---|---|---|
| `CHANGE_MAP` sb | `verified` | report + marker (`field/serverbound/change_test.go:158`) + evidence `field.serverbound.FieldChange.yaml` |
| `NPC_TALK` sb | `verified` | report + marker (`npc/serverbound/start_conversation_test.go:12`) + evidence |
| `CHANGE_MAP_SPECIAL` sb | `verified` | report + marker (`portal/serverbound/script_test.go:11`); tier-0, no evidence needed |
| `NPC_TALK` cb | `verified` | report + marker (`npc/clientbound/conversation_test.go` NpcNpcConversation, ida=0x6de0f0) + evidence |

`matrix --check` must exit 0. If a cell lands `incomplete` with
`byte-test marker present but no fresh evidence record`, the record's
`decompile_sha256` drifted — re-pin per playbook §7, do not delete the check.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/registry/gms_v95.yaml docs/tasks/task-146-v95-packet-verification-batch/fname-promotions.md docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "fix(packets): promote gms_v95 primary fnames to export-resident senders"
```

---

## Task 3: W0c — add `packet:` links for the four report-less ops

`grade.go:118-166` gives a second promotion path: with no report, the registry's
`packet:` field is the only way grading can find the marker + evidence for a
cell. Four in-scope ops will never have a report (their primary sender is absent
from the committed export and re-harvesting it is forbidden, playbook §10), so
they need the link.

### Files

- `docs/packets/registry/gms_v95.yaml` — entries at lines 2548 (`STORAGE` sb), 2391 (`ENTER_CASHSHOP` sb), 2987 (`MULTI_CHAT` sb), 148 (`CHECK_SPW_RESULT` cb)
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Patterns to copy: `docs/packets/registry/gms_v95.yaml:225-239` — `packet:` is a
sibling key at the same indent as `fname:`, placed immediately after `fname:` /
`fname_alts:` and before `provenance:`.

- [ ] **Step 1: Add the four `packet:` keys**

| Op × dir | `packet:` value |
|---|---|
| `STORAGE` sb (line 2548) | `storage/serverbound/StorageOperation` |
| `ENTER_CASHSHOP` sb (line 2391) | `cash/serverbound/CashShopEntry` |
| `MULTI_CHAT` sb (line 2987) | `chat/serverbound/ChatMulti` |
| `CHECK_SPW_RESULT` cb (line 148) | `login/clientbound/LoginPicResult` |

Each value is `qualifiedWriterName(pkg, struct)` = TitleCase(pkg) + struct name
(`run.go:223-228`), confirmed during planning against the actual Go struct
names: `cash/serverbound.ShopEntry`, `chat/serverbound.Multi`,
`login/clientbound.PicResult`, `storage/serverbound.Operation`. Set
`provenance: manual` and add a one-line `note:` on each explaining that the
primary fname is absent from the export.

Do **not** change any opcode here.

- [ ] **Step 2: Regenerate and check**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Expected `gms_v95` op-row states after this task:

- `STORAGE` sb → **`verified`**. `storage/serverbound/StorageOperation` already
  has marker (`libs/atlas-packet/storage/serverbound/operation_test.go:9`,
  ida=`0x76a990`) and evidence
  (`docs/packets/evidence/gms_v95/storage.serverbound.StorageOperation.yaml`).
  `grade.go:198-222` no-report path: marker + fresh evidence ⇒ verified.
- `ENTER_CASHSHOP` sb, `MULTI_CHAT` sb, `CHECK_SPW_RESULT` cb → still
  `incomplete`, note `no audit report` (no marker yet). Tasks 12–14 supply
  marker + evidence and flip them. **This is expected — do not treat it as a
  failure.**

`matrix --check` must exit 0.

- [ ] **Step 3: Commit**

```bash
git add docs/packets/registry/gms_v95.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "feat(packets): link four report-less gms_v95 ops to their codecs via packet:"
```

---

## Task 4: W0d — resolve the `LOGIN_AUTH` clientbound v95 seed

`docs/packets/registry/gms_v95.yaml:4` is an unresolved csv-import seed:

```yaml
- op: LOGIN_AUTH
  direction: clientbound
  opcode: 0
  fname: ""
  provenance: csv-import
```

Per playbook ("Is this cell `n-a`?" rule 1) an empty seed is
absence-of-evidence, not evidence-of-absence — so the PRD's listing of
`LoginAuth` under W4 is not yet earned. `gms_v83/v84/v87` carry the op at
opcode 23 as `CLogin::OnEnableSPWResult`; v95 has `CHECK_SPW_RESULT` at 27.
Resolve it, then either fill it in or delete the entry (deletion is how absence
is expressed — `opregistry.go` `Applicability`: not present ⇒ `Absent` ⇒ ⬜).

### Files

- `docs/packets/registry/gms_v95.yaml` — the `LOGIN_AUTH` clientbound entry, line 4
- `docs/tasks/task-146-v95-packet-verification-batch/login-auth-resolution.md` — **new file**; the positive proof
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

- [ ] **Step 1: Resolve in the IDB**

Database `ecc757f4`. Anchors, per playbook rule 2 — never a failed name search:

1. `CLogin::OnCheckSPWResult` is at `0x5d23f0` (mangled
   `?OnCheckSPWResult@CLogin@@IAEXAAVCInPacket@@@Z`). Find the login packet
   dispatcher that reaches it and enumerate its sibling arms — an
   `OnEnableSPWResult`-shaped arm at opcode 23 either exists or does not.
2. Cross-check the `StringPool` / `CLoginUtilDlg::Error` ids the SPW/PIC UI
   reads (`OnCheckSPWResult` uses error id 93).
3. MANDATORY sibling cross-check (playbook rule 3): the serverbound
   `CHECK_SPW`/`ENABLE_SPW` send side. If the client can *send* the request,
   the result arm exists somewhere.

- [ ] **Step 2: Apply the outcome**

- **Found** → set the real `opcode:` and `fname:`, `provenance: manual`, add an
  `ida.address`, and — if the fname is absent from the export — a `packet:`
  link. The cell then grades on its own merits.
- **Genuinely absent** → delete the entry from `gms_v95.yaml`. The op row's v95
  cell becomes `⬜ n-a` by absence, which is the intended mechanism.

Either way, write the anchors, addresses and reasoning into
`login-auth-resolution.md`. **Do not touch `gms_v92.yaml`**, which carries the
identical seed — out of scope (§ Follow-ups).

- [ ] **Step 3: Regenerate, check, commit**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Exit 0 required. Note that `matrix --check` runs the n-a-consistency gate
internally (`na_consistency.go`) — there is no standalone `na-consistency`
subcommand, so PRD §4.5's separate acceptance line is satisfied here.

```bash
git add docs/packets/registry/gms_v95.yaml docs/tasks/task-146-v95-packet-verification-batch/login-auth-resolution.md docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "fix(packets): resolve gms_v95 LOGIN_AUTH clientbound registry seed"
```

---

## Task 5: packet-audit — stop `NPC_TALK_MORE` from swallowing `NpcAskSlideMenuConversationDetail`

See §0/C5. The codec is fully verified on v95 (marker + evidence + report); its
`❌` is a report-consumption artifact of `build.go` indexing reports by
`baseFName`. `legacyConsumedSiblingWriters` is the file's own documented remedy,
and its `protectedWriters` gate makes it safe: a listed sibling escapes the skip
**only when `gradeSubStructCell` independently reaches `StateVerified`**
(`build.go:255-270`) — it is not a force-promote.

### Files

- `tools/packet-audit/internal/matrix/build.go` — extend `legacyConsumedSiblingWriters` (map literal begins at line 113) and its comment block
- `tools/packet-audit/internal/matrix/build_test.go` — new test case
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Patterns to copy: `tools/packet-audit/internal/matrix/build.go:40-112` (the
NOTE_ACTION and USE_CASH_ITEM comment blocks — match their evidence density) and
the existing `opKey("NOTE_ACTION", opregistry.DirServerbound)` map entry.

Module root: `tools/packet-audit`.

- [ ] **Step 1: Write the failing test**

Add to `tools/packet-audit/internal/matrix/build_test.go`. Copy the setup shape
from the existing NOTE_ACTION sub-struct test in that file (locate it by
grepping `NoteOperationDiscard`); it builds a synthetic `matrix.Inputs` rather
than reading the repo.

`TestAskSlideMenuEscapesNpcTalkMoreConsumption` — table-driven, one `t.Run` per
case:

| case | inputs | expected sub-struct cell for `npc/clientbound/NpcAskSlideMenuConversationDetail` |
|---|---|---|
| `verified sibling escapes` | registry op `NPC_TALK_MORE`/serverbound with `fname: CScriptMan::OnAskSlideMenu`; report `NpcAskSlideMenuConversationDetail` with `IDAName: CScriptMan::OnAskSlideMenu#AskSlideMenu`; marker found; evidence present and `Fresh: true`; `Tier1: true` | `State: StateVerified`, `Note: ""`, `Opcode: -1` |
| `unverified sibling stays suppressed` | same, but evidence absent | `State: StateIncomplete`, `Note: "no audit report"` (the gap-fill note — the protected bypass must NOT fire) |
| `stale evidence stays suppressed` | same, but `Fresh: false` | `State: StateIncomplete`, `Note: "no audit report"` |

The second and third cases are the point of the test: they pin the
`protectedWriters` gate, so a future edit cannot turn the allowlist into a
force-promote.

- [ ] **Step 2: Run the test and watch it fail**

```
cd tools/packet-audit && go test ./internal/matrix/ -run TestAskSlideMenuEscapesNpcTalkMoreConsumption -v
```

Expected: FAIL on case 1 — actual `StateIncomplete` / `"no audit report"`.
Cases 2 and 3 should already pass.

- [ ] **Step 3: Add the allowlist entry**

In `legacyConsumedSiblingWriters` add:

```go
	opKey("NPC_TALK_MORE", opregistry.DirServerbound): {
		"NpcAskSlideMenuConversationDetail": true,
	},
```

Precede it with a comment block in the style of the two above it, stating: the
op's v95 registry primary `fname` is `CScriptMan::OnAskSlideMenu`
(`docs/packets/registry/gms_v95.yaml:2519-2522`), which is exactly
`baseFName("CScriptMan::OnAskSlideMenu#AskSlideMenu")`, the clientbound detail
writer's own `IDAName`; the sibling `NpcSayImageConversationDetail` escapes only
because `CScriptMan::OnSayImage` is an `fname_alt` rather than the primary; the
writer carries its own marker (`conversation_test.go:25`, ida=`0x6dbe50`),
pinned evidence, and report on `gms_v95`; and per the `protectedWriters` gate
this only un-suppresses a cell that independently grades `verified`.

**Scope check before committing:** list every `(version, cell)` this flips. Only
`npc/clientbound/NpcAskSlideMenuConversationDetail × gms_v95` may change. If any
other version's cell flips, that version has not been per-cell verified and the
entry must be narrowed — the NOTE_ACTION comment records exactly this deferral
discipline.

- [ ] **Step 4: Run tests, regenerate, gate**

```
cd tools/packet-audit && go test ./...
```

then from the worktree root:

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit dispatcher-lint
```

All exit 0. `npc/clientbound/NpcAskSlideMenuConversationDetail × gms_v95` must
read `verified`.

- [ ] **Step 5: Commit**

```bash
git add tools/packet-audit/internal/matrix/build.go tools/packet-audit/internal/matrix/build_test.go docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "fix(packet-audit): let NpcAskSlideMenu detail grade from its own v95 evidence"
```

---

## Task 6: W1 — route eleven missing handlers/writers in the v95 template

Eleven of the twelve W1 entries land here. `StorageOperationHandle` is
**excluded**: Task 8 generates it from its dispatcher doc via
`packet-audit operations` (`operations.go:166-190`), which is the gated path.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — the only edited file; `.socket.handlers` (154 entries) and `.socket.writers` (239 entries)
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Read-only references (do not edit):
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json` for
validator / `services` shape only — **never for an opcode**;
`services/atlas-channel/atlas.com/channel/main.go` and
`services/atlas-login/atlas.com/login/main.go` for the registration check.

- [ ] **Step 1: Confirm every symbol has a live registration**

Design Finding 4 says all twelve are reachable; confirmed during planning:

| Entry | Kind | Registered at |
|---|---|---|
| `CashShopEntryHandle` | handler | `channel/main.go:933` |
| `MapChangeHandle` | handler | `channel/main.go:930` |
| `NPCStartConversationHandle` | handler | `channel/main.go:978` |
| `PortalScriptHandle` | handler | `channel/main.go:927` |
| `CharacterMultiChatHandle` | handler | `channel/main.go:973` — Go const is `chatSB.CharacterChatMultiHandle`, whose **value** is the string `"CharacterMultiChatHandle"` (`libs/atlas-packet/chat/serverbound/multi.go:15`). The template carries the string. |
| `PartyInviteRejectHandle` | handler | `channel/main.go:972` |
| `CharacterAutoDistributeApHandle` | handler | `channel/main.go:984` |
| `PongHandle` | handler | `channel/main.go:1049`, `login/main.go:388` |
| `CashShopOpen` | writer | `channel/main.go:703` (`cashcb.CashShopOpenWriter`) |
| `NPCConversation` | writer | `channel/main.go:766` (`npccb.NpcConversationWriter`) |
| `PicResult` | writer | `login/main.go:353` (`loginCB.PicResultWriter`) — login-only, **not** registered in channel |

If any is missing, that is a blocker to surface, not to stub.

- [ ] **Step 2: Insert the entries, sorted by `opCode`**

Both arrays are sorted ascending by numeric `opCode`; insert in place, never
append. Preserve LF line endings and the file's existing indent style.
Handler key order: `opCode, validator, handler, fname, options, services`
(omit `options` where there is none). Writer key order: `opCode, writer, fname,
options, services`.

| Entry | Kind | opCode | validator | services | fname | Source |
|---|---|---|---|---|---|---|
| `CashShopEntryHandle` | handler | `0x2B` | `LoggedInValidator` | `["channel"]` | `CWvsContext::SendMigrateToShopRequest` | `ENTER_CASHSHOP` sb 43 |
| `MapChangeHandle` | handler | `0x29` | `LoggedInValidator` | `["channel"]` | `CField::SendTransferFieldRequest` | `CHANGE_MAP` sb 41 (post-Task-2 primary) |
| `NPCStartConversationHandle` | handler | `0x3F` | `LoggedInValidator` | `["channel"]` | `CUserLocal::TalkToNpc` | `NPC_TALK` sb 63 |
| `CharacterAutoDistributeApHandle` | handler | **`0x63`** | `LoggedInValidator` | `["channel"]` | `CWvsContext::SendAbilityUpRequest` | `AUTO_DISTRIBUTE_AP` sb **99**, per Task 1 |
| `PortalScriptHandle` | handler | `0x70` | `LoggedInValidator` | `["channel"]` | `CUserLocal::CheckPortal_Collision` | `CHANGE_MAP_SPECIAL` sb 112 |
| `CharacterMultiChatHandle` | handler | `0x8C` | `LoggedInValidator` | `["channel"]` | `CUIStatusBar::SendGroupMessage` | `MULTI_CHAT` sb 140 |
| `PartyInviteRejectHandle` | handler | `0x92` | `LoggedInValidator` | `["channel"]` | `CWvsContext::OnPartyResult` | `PARTY_RESULT` sb 146 — see Step 3 |
| `PongHandle` | handler | `0x19` | `NoOpValidator` | `["login","channel"]` | `CClientSocket::OnAliveReq` | `PONG` sb 25 — **replaces** the existing `NoOpHandler` entry |
| `PicResult` | writer | `0x1B` | — | `["login"]` | `CLogin::OnCheckSPWResult` | `CHECK_SPW_RESULT` cb 27 |
| `CashShopOpen` | writer | `0x8F` | — | `["channel"]` | `CStage::OnSetCashShop` | `SET_CASH_SHOP` cb 143 |
| `NPCConversation` | writer | `0x16B` | — | `["channel"]` | `CScriptMan::OnScriptMessage` | `NPC_TALK` cb 363 — `options.messageType` is added in Task 9, not here |

`PongHandle` at `0x19`: the current entry
(`template_gms_95_1.json:199-207`) is
`{"opCode":"0x19","validator":"NoOpValidator","handler":"NoOpHandler","fname":"CClientSocket::OnAliveReq","services":["login","channel"]}`
— byte-for-byte the shape v83 gives `PongHandle` at `0x18`. Change `handler` to
`PongHandle`; leave everything else.

Collision check before writing: `0x19` (the NoOpHandler stub, intentionally
replaced) is the only occupied slot among these. `0x63` and `0x92` are free
(planning confirmed `0x62` = `CharacterDistributeApHandle`, `0x91` =
`PartyOperationHandle`, `0x95` = `GuildOperationHandle`). Re-verify
mechanically before editing.

- [ ] **Step 3: `PartyInviteRejectHandle` — confirm the send site before routing**

This one needs proof, and the design's assertion is not sufficient on its own.
The facts: `DENY_PARTY_REQUEST` sb — the op that carries invite-reject on
v48–v84 and that v83 routes at `0x7D` — is **`n-a` on gms_v87/v92/v95/jms_v185**
in `status.json`. On v95 the reject rides `PARTY_RESULT` sb 146, which is already
`verified`, and its report
`docs/packets/audits/gms_v95/PartyInviteReject.json` cites
`IDAName: CWvsContext::OnPartyResult#InviteReject` @`0xa10ab0` against
`libs/atlas-packet/party/serverbound/invite_reject.go`.

Confirm in the IDB (database `ecc757f4`) that the v95 client builds
`COutPacket(&pkt, 146)` for the party-decline path — anchor on the `COutPacket`
ctor xrefs near `0xa10ab0`, or the playbook §10 byte signature with op
`0x92`. **If confirmed**, route at `0x92` as tabled. **If not**, drop
`PartyInviteRejectHandle` from this task, record the negative proof in
`docs/tasks/task-146-v95-packet-verification-batch/v95-na-proof.md` (Task 16
creates it), and note the PRD §4.1 deviation. Do not route an opcode you could
not confirm.

- [ ] **Step 4: Regenerate and gate**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Exit 0 required. Routing is load-bearing for serverbound grading (playbook §9:
a serverbound cell needs the op routed in that version's seed template), so
watch for the `routedElsewhere && !routed` conflict class **disappearing**, not
appearing. Confirm no non-v95 template file is in `git status`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "feat(configurations): route eleven missing handlers/writers in the gms_v95.1 template"
```

---

## Task 7: W3a — handler-keyed dispatcher docs for party, guild, buddy

The five existing docs named in design §4.1 (`party.yaml`, `guild.yaml`,
`buddy.yaml`, `messenger_operation.yaml`, `guild_bbs.yaml`) are all
`writer:`-keyed **clientbound** result tables. The W3 gap is the **serverbound
handler** table — a different enumeration, derived from the client's *send*
sites, which nothing in the repo owns today. That is why
`operations --check` currently reports "0 absent-writer notes" while the
templates carry no table: there is no doc to check against.

Three docs here, three in Task 8, to keep each within one derivation sitting.

### Files

- `docs/packets/dispatchers/party_operation_handle.yaml` — **new file**
- `docs/packets/dispatchers/guild_operation_handle.yaml` — **new file**
- `docs/packets/dispatchers/buddy_operation_handle.yaml` — **new file**
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — written by the generator, not by hand
- `docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md` — **new file**; per-mode IDA citations

Patterns to copy: `docs/packets/dispatchers/duey_action.yaml` (the
`handler:` + `op:` + `direction: serverbound` + `operations:` shape) and
`docs/packets/dispatchers/cash_shop_operation_handle.yaml:1-40` (the
"enumerated by construction, unnamed arms omitted rather than guessed" comment
discipline — match its evidence density).

Read-only:
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`'s
`operations` blocks are a **key-vocabulary** reference only. No v83 mode value
is copied.

Module root for the generator: repo root (`go run ./tools/packet-audit ...`).

- [ ] **Step 1: Enumerate each handler's v95 mode bytes in the IDB**

Database `ecc757f4`. For each opcode below, enumerate every client send site
that builds `COutPacket(&pkt, <opcode>)` and read its leading `Encode1(mode)`.
Per `cash_shop_operation_handle.yaml`'s own comment, the "mode switch" is a
**server-side** construct — the client has no single builder — so enumerate by
exhaustive byte search for the opcode push (`68 <op> 00 00 00` / `6A <op>`)
across the owning UI region, not by name search. That is what makes the table
complete by construction.

| Doc | handler | v95 opCode | template line today |
|---|---|---|---|
| `party_operation_handle.yaml` | `PartyOperationHandle` | `0x91` | 1077 |
| `guild_operation_handle.yaml` | `GuildOperationHandle` | `0x95` | 1086 |
| `buddy_operation_handle.yaml` | `BuddyOperationHandle` | `0x99` | 1122 |

Record every arm — mode byte, send-site address, build-site address, and the
Atlas key it maps to — in `v95-option-tables.md`. **An arm whose Atlas key is
unknown is omitted, never guessed** (the `cash_shop_operation_handle.yaml`
precedent); say so explicitly in the doc's header comment.

- [ ] **Step 2: Write the three docs**

Each takes this shape (`party_operation_handle.yaml` shown; the schema is
`dispatcherDoc` at `tools/packet-audit/cmd/operations.go:38-81`):

```yaml
# PartyOperationHandle — serverbound PARTY_OPERATION mode table (gms_v95 only).
#
# SOURCE OF TRUTH for this handler's options.operations map.
#
# gms_v95 ONLY. Declaring another version here would make `packet-audit
# operations` rewrite that version's hand-authored table with an un-derived
# carry-over — exactly the v83->v95 drift this task exists to catch, and a
# violation of "no non-v95 template is modified". Backfilling IDA-derived
# columns for v83/v84/v87/v92/jms_v185 is filed as a follow-up.
#
# ALL-OR-NOTHING: expectedTable reports every template key absent from this
# file as EXTRA once a version has at least one declared key, so gms_v95 must
# be enumerated COMPLETELY. Per-arm addresses: v95-option-tables.md.

handler: PartyOperationHandle
validator: LoggedInValidator
services: [channel]
fname: CField::SendCreateNewPartyMsg
op: PARTY_OPERATION
direction: serverbound
operations:
  - { key: <KEY>, modes: { gms_v95: <n> } }
  # ... one line per enumerated arm
```

`direction: serverbound` under a `handler:` key is required to opt out of the
FAM-CAP check (`duey_action.yaml:25-27`). Keep `fname:` as the registry primary
for that op. Do **not** add an `opcodes:` block to these three — the handlers
are already routed, so entry creation is not wanted (`operations.go:166-190`
only creates when the template entry is missing *and* `opcodes[vk]` is declared).

- [ ] **Step 3: Generate, then gate**

```
go run ./tools/packet-audit operations
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
```

`operations` writes `options.operations` into the three v95 template entries;
`--check` must then exit 0 with no `drift`, `missing` or `extra`. Note that an
`operations note (writer absent): ...` line is diagnostic only and never fails
the check (`operations.go:170-181`, exit code at `:246`).

Confirm with `git diff --stat` that **only** `template_gms_95_1.json` changed
among the templates.

- [ ] **Step 4: Regenerate the matrix and commit**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

```bash
git add docs/packets/dispatchers/party_operation_handle.yaml docs/packets/dispatchers/guild_operation_handle.yaml docs/packets/dispatchers/buddy_operation_handle.yaml services/atlas-configurations/seed-data/templates/template_gms_95_1.json docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "feat(packets): gms_v95 serverbound mode tables for party/guild/buddy handlers"
```

---

## Task 8: W3a — handler-keyed dispatcher docs for messenger, guild BBS, storage

Same procedure as Task 7. `storage_operation_handle.yaml` differs: it also
declares `opcodes:`, so the generator **creates** the template handler entry
(`operations.go:166-190` → `addEntry` at `:578-628`, inserted by `insertSorted`
at `:635-666`). That is why `StorageOperationHandle` was held out of Task 6.

### Files

- `docs/packets/dispatchers/messenger_operation_handle.yaml` — **new file**
- `docs/packets/dispatchers/guild_bbs_handle.yaml` — **new file**
- `docs/packets/dispatchers/storage_operation_handle.yaml` — **new file**
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — written by the generator
- `docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md` — **new file** if absent (created in Task 7); append

Patterns to copy: the three docs written in Task 7;
`docs/packets/dispatchers/cash_shop_coupon_code.yaml` (the minimal
`handler + validator + services + fname + op + direction + opcodes` shape, for
the `storage` doc's entry-creating fields).

- [ ] **Step 1: Enumerate the v95 modes**

| Doc | handler | v95 opCode | template today |
|---|---|---|---|
| `messenger_operation_handle.yaml` | `MessengerOperationHandle` | `0x8F` | routed, line 1010 |
| `guild_bbs_handle.yaml` | `GuildBBSHandle` | `0xB3` | routed, line 1220 |
| `storage_operation_handle.yaml` | `StorageOperationHandle` | `0x43` | **not routed** |

Same exhaustive-byte-search discipline as Task 7 Step 1. For storage the four
Atlas arms are already known and verified — `storage/serverbound/operation.go`,
`operation_meso.go`, `operation_retrieve_asset.go`, `operation_store_asset.go`,
whose v95 send sites are `CTrunkDlg::SendGetMoneyRequest` @`0x7688e0`,
`CTrunkDlg::SendGetItemRequest` @`0x769e00`, `CTrunkDlg::SendPutItemRequest`
@`0x768570` — so the mode bytes come from those senders' leading `Encode1`,
plus whatever further arms the byte search turns up.

Append every arm with its addresses to `v95-option-tables.md`.

- [ ] **Step 2: Write the three docs**

`messenger_operation_handle.yaml` and `guild_bbs_handle.yaml` follow Task 7's
shape exactly (no `opcodes:` block — already routed).

`storage_operation_handle.yaml` additionally carries the entry-creating fields:

```yaml
handler: StorageOperationHandle
validator: LoggedInValidator
services: [channel]
fname: CTrunkDlg::SetRet
op: STORAGE
direction: serverbound
opcodes:
  gms_v95: "0x43"
operations:
  - { key: <KEY>, modes: { gms_v95: <n> } }
```

Note the deliberate asymmetry with Task 3: the registry keeps
`fname: CTrunkDlg::SetRet` (it is the CSV's name for the op and other versions
key on it) and reaches its codec through `packet:`; the dispatcher doc's
`fname:` mirrors the registry primary.

- [ ] **Step 3: Generate and gate**

```
go run ./tools/packet-audit operations
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit dispatcher-lint
```

Verify the generated `StorageOperationHandle` entry landed at `0x43` in
ascending position with key order `opCode, validator, handler, fname, options,
services`. Confirm no other template file changed.

- [ ] **Step 4: Regenerate the matrix and commit**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

```bash
git add docs/packets/dispatchers/messenger_operation_handle.yaml docs/packets/dispatchers/guild_bbs_handle.yaml docs/packets/dispatchers/storage_operation_handle.yaml services/atlas-configurations/seed-data/templates/template_gms_95_1.json docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "feat(packets): gms_v95 serverbound mode tables for messenger/BBS/storage handlers"
```

---

## Task 9: W3b — `messageType` for `NPCContinueConversationHandle` and `NPCConversation`

`options.messageType` is a flat `string → int` map with no generator and no
schema in `packet-audit` — hand-authored and ungated, so review is the only
protection. Derive every value from the v95 client.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — the `NPCContinueConversationHandle` handler entry at line 567 (`0x41`), and the `NPCConversation` writer entry added in Task 6 (`0x16B`)
- `docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md` — **new file** if absent (created in Task 7); append the per-value citations

Read-only: `libs/atlas-packet/npc/clientbound/conversation.go` lines 16-35 — the
`NpcConversationMessageType` string constants are the **key vocabulary**;
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`'s
14-key block is a shape reference only, no value copied.

- [ ] **Step 1: Derive the v95 `msgType` switch**

Database `ecc757f4`. `CScriptMan::OnScriptMessage` @`0x6de0f0` reads the common
header and switches on `msgType`. Enumerate every arm and its integer, and map
each to the `*ConversationDetail` struct that already encodes it. The 14 detail
structs are at `libs/atlas-packet/npc/clientbound/conversation.go` lines 112,
129, 158, 171, 190, 209, 238, 265, 290, 308, 340, 359, 380 and 399.

**Cross-check, and it is not optional:** thirteen of those structs already carry
a `gms_v95` marker whose `ida=` is the arm's own address —
`AskAvatar 0x6dcff0`, `AskBoxText 0x6dc9c0`, `AskMenu 0x6dce00`,
`AskNumber 0x6dcc00`, `AskPetAll 0x6ddbe0`, `AskPet 0x6dd6e0`,
`AskQuiz 0x9ffad0`, `AskSlideMenu 0x6dbe50`, `AskSpeedQuiz 0x9f1d50`,
`AskText 0x6dc790`, `AskYesNo 0x6dc5a0`, `Say 0x6dc110`, `SayImage 0x6dc310`
(markers in `libs/atlas-packet/npc/clientbound/conversation_test.go` lines
23-80). A `messageType` value that does not select the arm its detail struct is
fixtured against is **wrong** — reconcile before writing.

- [ ] **Step 2: Write both `options` blocks**

Insert an `options` object (the entries currently have **no** `options` key —
§0/C6) between `fname` and `services`:

```json
"options": {
  "messageType": {
    "SAY": 0, "ASK_YES_NO": 0, "ASK_TEXT": 0, "ASK_NUMBER": 0,
    "ASK_MENU": 0, "ASK_QUIZ": 0, "ASK_SPEED_QUIZ": 0,
    "ASK_AVATAR": 0, "ASK_MEMBER_SHOP_AVATAR": 0, "ASK_PET": 0,
    "ASK_PET_ALL": 0, "ASK_YES_NO_QUEST": 0, "ASK_BOX_TEXT": 0,
    "ASK_SLIDE_MENU": 0
  }
}
```

Every `0` above is a **placeholder for the value Step 1 derives** — the shape is
fixed, the values are not known at plan time and must not be guessed. Keys are
exactly the v83 key set unless the v95 switch shows an arm the v83 set lacks
(add it) or lacks an arm v83 has (omit it and say so in
`v95-option-tables.md`). **Every value is IDB-derived; none is copied from
v83.** If a v95 value happens to equal the v83 value, that is a finding to
record, not a shortcut to take.

- [ ] **Step 3: Gate and commit**

```
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit matrix --check
```

Both exit 0. `messageType` is outside `packet-audit`'s schema, so neither gate
validates its contents — the citations in `v95-option-tables.md` are the
evidence, and code review is the check.

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md
git commit -m "feat(configurations): IDB-derived gms_v95 NPC conversation messageType table"
```

---

## Task 10: W3c — `failedReasonCodes` for `AuthPermanentBan` and `AuthTemporaryBan`

Both writers are arms of `CLogin::OnCheckPasswordResult` (`LOGIN_STATUS` cb,
verified on v95) and share one table, as on v83.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — writer entries `AuthPermanentBan` (line 2476, `0x00`) and `AuthTemporaryBan` (line 2492, `0x00`)
- `docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md` — **new file** if absent (created in Task 7); append

Read-only: the v95 `AuthLoginFailed` writer entry (`0x00`) in the same file
**already carries a populated `failedReasonCodes` block** — it is the in-version
shape reference and a cross-check on the value space.
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`'s
`AuthPermanentBan` block is a key-vocabulary reference only.

- [ ] **Step 1: Derive the reason switch**

Database `ecc757f4`. `CLogin::OnCheckPasswordResult` is at **`0x5dc600`**
(mangled `?OnCheckPasswordResult@CLogin@@IAEXAAVCInPacket@@@Z`; it is **absent
from the committed export**, so this is a live decompile). Enumerate every
reason-code arm with its integer and the `StringPool` / `CLoginUtilDlg::Error`
id it raises, and map each to the Atlas key.

Cross-check the result against the `AuthLoginFailed` block already in the v95
template: the two must agree wherever they overlap. A disagreement means one of
them is wrong — resolve it in the IDB and say which, in
`v95-option-tables.md`. Do not silently pick one.

- [ ] **Step 2: Write the identical block into both writer entries**

Insert `"options": { "failedReasonCodes": { ... } }` between `fname` and
`services` on both. Keys follow the v83/v95 vocabulary (`BANNED`,
`DELETED_OR_BLOCKED`, `INCORRECT_PASSWORD`, `NOT_REGISTERED`, `SYSTEM_ERROR_1`,
`ALREADY_LOGGED_IN`, `SYSTEM_ERROR_2`, `SYSTEM_ERROR_3`,
`TOO_MANY_CONNECTIONS`, `AGE_LIMIT`, `UNABLE_TO_LOG_ON_AS_MASTER_AT_IP`,
`WRONG_GATEWAY`, `PROCESSING_REQUEST`, `ACCOUNT_VERIFICATION_NEEDED`,
`WRONG_PERSONAL_INFORMATION`, `ACCOUNT_VERIFICATION_NEEDED_2`,
`LICENSE_AGREEMENT`, `MAPLE_EUROPE_NOTICE`, `FULL_CLIENT_NOTICE`); every
**value** is IDB-derived. An arm the v95 switch has and the key list lacks gets
a new key, recorded in `v95-option-tables.md`.

- [ ] **Step 3: Gate and commit**

```
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit matrix --check
```

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md
git commit -m "feat(configurations): IDB-derived gms_v95 auth ban failedReasonCodes table"
```

---

## Task 11: W3d — the movement `types` array for four v95 writers

The highest-severity item in the task, and the only one with no matrix cell.
`options.types` is **not** a map: it is an **index-addressed array** of
`{Name, Type}` objects. `libs/atlas-packet/model/movement.go` lines 384-397:

```go
func resolveMovementPathAttr(attr byte, options map[string]interface{}) (string, string, bool) {
	genericCodes, ok := options["types"]
	if !ok { return "NOT_FOUND", "DEFAULT", false }
	codes, ok := genericCodes.([]interface{})
	if !ok { return "NOT_FOUND", "DEFAULT", false }
	if len(codes) == 0 || int(attr) >= len(codes) { return "NOT_FOUND", "DEFAULT", false }
	theType, ok := codes[attr].(map[string]interface{})
	if !ok { return "NOT_FOUND", "DEFAULT", false }
	return theType["Name"].(string), theType["Type"].(string), true
}
```

The fragment's attribute byte indexes the array directly and `Type` selects the
decode shape. With no table **every** fragment resolves `NOT_FOUND`/`DEFAULT` —
v95 movement is misparsing today. Index order **is** the wire contract; an
off-by-one is a silent misparse, not an error.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — writer entries `PetMovement` (line 3668, `0xC9`), `CharacterMovement` (line 3732, `0xD2`), `MoveMonster` (line 4119, `0x11F`), `NPCAction` (line 4309, `0x13A`)
- `libs/atlas-packet/character/clientbound/movement_test.go` — feed the real table (see Step 3)
- `libs/atlas-packet/monster/clientbound/movement_test.go` — same
- `libs/atlas-packet/pet/clientbound/movement_test.go` — same
- `libs/atlas-packet/test/movement_types.go` — **new file**; the single shared v95 table helper
- `docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md` — **new file** if absent (created in Task 7); per-index IDA citations

Read-only: `libs/atlas-packet/model/movement.go` lines 377-463
(`resolveMovementPathAttr` and its callers) — the consumers; do not change them.
`services/atlas-configurations/seed-data/templates/template_gms_83_1.json`'s
35-entry `CharacterMovement` table is a `{Name, Type}` **vocabulary** reference
only. **No v83 index is copied.**

Module root: `libs/atlas-packet`.

- [ ] **Step 1: Derive the array from the v95 client**

Database `ecc757f4`. The four receive handlers, all resolved during planning:

| Writer | Client function | v95 address |
|---|---|---|
| `CharacterMovement` | `CUserRemote::OnMove` | `0x948a80` |
| `MoveMonster` | `CMob::OnMove` | `0x6521e0` |
| `PetMovement` | `CPet::OnMove` | `0x69fb60` |
| `NPCAction` | `CNpc::OnMove` | `0x678060` |

Descend from each into the shared move-path decode (the `CVecCtrl`/`CMovePath`
attribute switch) and enumerate every attribute value with its decode shape.
The array's index **is** the attribute byte, so a value the switch does not use
still occupies its slot — fill gaps rather than compacting. Record every index
with its address and switch arm in `v95-option-tables.md`.

If the four handlers do not share one attribute table on v95, they get four
different arrays — say so explicitly rather than assuming a single table.

- [ ] **Step 2: Write the array into all four writer entries**

`"options": { "types": [ {"Name": "...", "Type": "..."}, ... ] }`, inserted
between `fname` and `services` (`writers` entries carry no `validator`).

- [ ] **Step 3: Couple the fixtures to the template so a future desync fails `go test`**

There is no CI gate for `types` — this coupling is the substitute, and it is the
mitigation the design promised. Today:

- `libs/atlas-packet/character/clientbound/movement_test.go` lines 31-38 define a
  local `normalTypesOptions()` returning a **one-entry** stub table
  (`{"Name":"NORMAL","Type":"NORMAL"}`), used at line 69.
- `libs/atlas-packet/monster/clientbound/movement_test.go` and
  `libs/atlas-packet/pet/clientbound/movement_test.go` pass an **empty**
  `model.Movement{}` and never exercise a `types` table at all.

Replace the stub with the real v95 array, read from **one** shared source so a
template edit that desyncs the array fails the build:

1. Create `libs/atlas-packet/test/movement_types.go` exporting
   `MovementTypesV95() map[string]interface{}` that returns
   `{"types": []interface{}{ {"Name": ..., "Type": ...}, ... }}` with the Step 1
   array. Copy the literal shape from
   `libs/atlas-packet/character/clientbound/movement_test.go` lines 31-38 and the
   package idiom from `libs/atlas-packet/test/roundtrip.go`.
2. Point `normalTypesOptions()` at `pt.MovementTypesV95()`, and pass it in the
   monster and pet movement tests instead of the empty `model.Movement{}`.
3. Add one assertion per package that a movement element whose attribute byte is
   the **highest index in the array** still resolves — the expected `Name`/`Type`
   from the array, not `NOT_FOUND`/`DEFAULT`. That is the assertion an
   off-by-one or truncated array fails.

Do **not** change the existing v95 byte-golden expectations in those three files
(`character` marker at `movement_test.go:43` ida=`0x948a80`, `monster` at
`movement_test.go:13` ida=`0x6521e0`, `pet` at `movement_test.go:11`
ida=`0x69fb60` — all already verified). If feeding a real `types` table changes a
golden byte, that is a wire finding: stop, and report it before touching the
golden.

- [ ] **Step 4: Run tests and gate**

```
cd libs/atlas-packet && go test -race ./character/... ./monster/... ./pet/... ./model/... ./test/... && go vet ./...
```

then from the worktree root:

```
go run ./tools/packet-audit operations --check
go run ./tools/packet-audit matrix --check
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_gms_95_1.json libs/atlas-packet/test/movement_types.go libs/atlas-packet/character/clientbound/movement_test.go libs/atlas-packet/monster/clientbound/movement_test.go libs/atlas-packet/pet/clientbound/movement_test.go docs/tasks/task-146-v95-packet-verification-batch/v95-option-tables.md
git commit -m "feat(configurations): IDB-derived gms_v95 movement types tables, pinned by fixtures"
```

---

## Task 12: W2 row 8 — verify `CHECK_SPW_RESULT` clientbound (`LoginPicResult`)

Do this before the other two full verifications: the client side is already
decompiled (§0/C3), the fixture is one byte, and it exercises the whole
`packet:` + tier-0-evidence path end to end at minimum cost.

Route: `packet:` (Task 3). Tier: **0**. `CLogin::OnCheckSPWResult` is absent
from the committed export, so no report exists and none can be generated
without a harvest — the cell can only promote via marker + fresh evidence
(`grade.go:198-222`), which is the documented §7 exception. Say so in the
record's `category`.

### Files

- `libs/atlas-packet/login/clientbound/pic_result_test.go` — add the byte-golden test and the marker
- `docs/packets/evidence/gms_v95/login.clientbound.LoginPicResult.yaml` — **new file**, written by `evidence pin`
- `docs/packets/ida-exports/gms_v95.json` — **surgical splice of one entry only**, if Step 4 requires it
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Read-only: `libs/atlas-packet/login/clientbound/pic_result.go` lines 19-31 — the
codec already matches the client; do not change it.

Patterns to copy: `libs/atlas-packet/login/clientbound/pic_result_test.go` lines
1-18 (existing round-trip; keep it) and
`libs/atlas-packet/login/clientbound/auth_login_failed_test.go` for the
byte-golden assertion shape in this package.

Module root: `libs/atlas-packet`.

- [ ] **Step 1: Re-confirm the client read order**

Decompile `0x5d23f0` on database `ecc757f4`. Expected, from planning:

```c
void __thiscall CLogin::OnCheckSPWResult(CLogin *this, CInPacket *iPacket)
{
  CInPacket::Decode1(iPacket);                     /*0x5d23f7*/
  CLoginUtilDlg::Error(93, &this->m_pChildModal);  /*0x5d2405*/
  this->m_bRequestSent = 0;                        /*0x5d240d*/
}
```

One `Decode1`, no further reads. `pic_result.go` line 22 writes exactly
`w.WriteByte(0)` — they agree, so **no wire fix** (playbook §4 is satisfied). If
the live read differs from the above, stop and report before writing a fixture.

- [ ] **Step 2: Write the byte-golden test**

Append to `pic_result_test.go`. Keep `TestPicResultRoundTrip` as-is.

`TestPicResultByteOutputV95` — single case, no table (the packet has one field
and no version gate):

- tenant: `pt.Variants[3]` (GMS v95.1)
- construct: `PicResult{}`
- encode via `pt.Encode(t, ctx, input.Encode, nil)`
- expected bytes: **`[]byte{0x00}`** — exactly one byte, the `Decode1` the client
  reads at `0x5d23f7`, which `pic_result.go` line 22 writes as a literal `0`.
- assert length is 1 as well as content, so a future added field fails loudly.

Marker, immediately above the function:

```go
// packet-audit:verify packet=login/clientbound/LoginPicResult version=gms_v95 ida=0x5d23f0
```

Cite `0x5d23f7` in an inline comment on the byte.

- [ ] **Step 3: Run the test**

```
cd libs/atlas-packet && go test ./login/clientbound/ -run TestPicResultByteOutputV95 -v
```

It should **pass immediately** — the codec is already correct. That is the
expected outcome here, not a TDD violation: this task verifies existing wire, it
does not add behaviour. Confirm it fails if you flip the expected byte to `0x01`,
then flip it back.

- [ ] **Step 4: Pin the evidence record**

```
go run ./tools/packet-audit evidence pin --packet login/clientbound/LoginPicResult --version gms_v95 --ida "CLogin::OnCheckSPWResult" --category TIER1-FIXTURE
```

`--ida` must be the name **as it appears as a key in the export's `functions`
map**. `CLogin::OnCheckSPWResult` is **not** in
`docs/packets/ida-exports/gms_v95.json`, so this command is expected to fail
with "not in export". When it does:

- Do **not** re-run `packet-audit export` and do **not** overwrite the committed
  export (playbook §10).
- Per playbook §10, harvest to a temp file with
  `-prior-export "" -pending <roster> -descent-depth 12` and **surgically splice
  only the `CLogin::OnCheckSPWResult` entry** into
  `docs/packets/ida-exports/gms_v95.json`. Splicing one absent entry is the
  documented remedy; wholesale re-export is not.
- Then re-run `evidence pin`, open the written YAML, and hand-add:

```yaml
verifies:
  - libs/atlas-packet/login/clientbound/pic_result_test.go#TestPicResultByteOutputV95
```

Set `category` to record the tier-0 exception explicitly: this cell has no
report and cannot promote without the record, so the freshness liability is
accepted deliberately (design §2.3).

If the splice turns out to be impossible, that is a genuine blocker — surface
it, do not claim the cell.

- [ ] **Step 5: Regenerate and gate**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
```

`CHECK_SPW_RESULT` cb × `gms_v95` must read **`verified`**. Exit 0 on all.

- [ ] **Step 6: Commit test + evidence + matrix together** (playbook §8)

```bash
git add libs/atlas-packet/login/clientbound/pic_result_test.go docs/packets/evidence/gms_v95/login.clientbound.LoginPicResult.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(atlas-packet): verify CHECK_SPW_RESULT clientbound on gms_v95"
```

Add `docs/packets/ida-exports/gms_v95.json` to that `git add` only if a splice
was actually needed.

---

## Task 13: W2 row 2 — verify `MULTI_CHAT` serverbound (`ChatMulti`)

Route: `packet:` (Task 3). Tier: **1** (`status.json`) — so the evidence pin is
ordinary, not an exception (§0/C2). The v95 gate already exists in the codec
(§0/C4), so this is fixture + evidence + a `gates.yaml` row, not a wire fix.

### Files

- `libs/atlas-packet/chat/serverbound/multi_test.go` — add the v95 byte-golden test and marker
- `docs/packets/evidence/gms_v95/chat.serverbound.ChatMulti.yaml` — **new file**
- `docs/packets/gates.yaml` — add the v92/v95 boundary row
- `docs/packets/ida-exports/gms_v95.json` — **surgical splice of one entry only**, if Step 4 requires it
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Read-only: `libs/atlas-packet/chat/serverbound/multi.go` lines 53-56 and 71-77 —
the gate; `tools/packet-audit/cmd/run.go` lines 1775-1783 — the recorded wire,
whose comment is **derived and must not be trusted as evidence**.

Patterns to copy: `libs/atlas-packet/chat/serverbound/multi_test.go` lines 12-49
(`TestMultiByteOutputV79`, the byte-golden shape) and lines 115-143
(`TestMultiUpdateTimeGate`, which already asserts the v95 update-time prefix but
carries **no marker** — an unpinned assertion, not a packet-audit fixture).

Module root: `libs/atlas-packet`.

- [ ] **Step 1: Derive the v95 read order live**

`CUIStatusBar::SendGroupMessage` is **not resolvable by name** in the IDB (a
mangled-name guess failed) and is absent from the export. Find it by the
playbook §10 byte signature for a `COutPacket` build with opcode
**140 = `0x8C`** (`6A 8C 8D 8D ?? ?? ?? ?? E8`, or `68 8C 00 00 00`), then
structure-match against the named v79/v83 twin. **Naming it is producible work,
not a blocker** — name it in the IDB if it is unnamed.

Record the ordered encode list. `run.go` lines 1775-1783 claim, for v95:
`Encode4(updateTime)`, `Encode1(nChatTarget)`, `Encode1(nMemberCnt)`,
`loop Encode4(memberId) × n`, `EncodeStr(sText)`. Confirm or correct it live —
that comment is not evidence.

- [ ] **Step 2: Compare against Atlas and decide whether a wire fix is needed**

`multi.go` writes, under `hasUpdateTime` (GMS && major ≥ 95):
`updateTime`, then `chatType`, `recipientCount`, `recipients…`, `chatText`.
Planning's reading is that this **already matches**. If Step 1 shows otherwise,
the wire fix lands first, as its own commit with its own review (playbook §4),
before any fixture.

- [ ] **Step 3: Write the failing byte-golden test**

Append `TestMultiByteOutputV95` to `multi_test.go`, copying the setup shape from
`TestMultiByteOutputV79`.

| field | fixture value | expected bytes | cite |
|---|---|---|---|
| `updateTime` | `0x11223344` | `44 33 22 11` | the v95-only `Encode4` at the address found in Step 1 |
| `chatType` | `1` | `01` | `Encode1(nChatTarget)` |
| `recipientCount` | `2` | `02` | `Encode1(nMemberCnt)` |
| `recipients[0]` | `1000` | `E8 03 00 00` | loop `Encode4` |
| `recipients[1]` | `2000` | `D0 07 00 00` | loop `Encode4` |
| `chatText` | `"hi"` | `02 00 68 69` | `EncodeStr` (length-prefixed ASCII) |

Total 15 bytes. Assert the full slice **and** its length. Add a second case
using `pt.Variants[11]` (GMS v92) asserting the **same fixture minus the leading
four bytes** (11 bytes) — that is what pins the gate boundary and what
`gate-check` will look for. Confirm the string encoding against
`libs/atlas-socket/response` before fixing the `EncodeStr` bytes; correct the
table to the real encoding if it differs.

Marker above the v95 test, with the address found in Step 1:

```go
// packet-audit:verify packet=chat/serverbound/ChatMulti version=gms_v95 ida=<0xaddr>
```

- [ ] **Step 4: Run, pin evidence, add the gate row**

```
cd libs/atlas-packet && go test ./chat/serverbound/ -run TestMultiByteOutput -v
```

Then pin (tier-1, ordinary):

```
go run ./tools/packet-audit evidence pin --packet chat/serverbound/ChatMulti --version gms_v95 --ida "CUIStatusBar::SendGroupMessage" --category TIER1-FIXTURE
```

If it fails "not in export", splice that one entry per playbook §10 — same rule
as Task 12 Step 4. Use whatever name the function actually carries after Step 1.
Hand-add `verifies:` pointing at the new test.

Then add to `docs/packets/gates.yaml` (schema documented at the head of that
file):

```yaml
  - packet: chat/serverbound/ChatMulti
    direction: serverbound
    field: v95 leading Encode4(updateTime) (`GMS && >=95`)
    boundary: ">=95"
    lower_version_key: gms_v92
    upper_version_key: gms_v95
```

`gate-check` requires a **verified** fixture on **both** straddling versions. If
`gms_v92` is not verified for `ChatMulti` after this task, set
`expect: partial` with a `reason` naming the gap — **never fabricate the v92
fixture to satisfy the gate**.

- [ ] **Step 5: Regenerate and gate**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit gate-check --check
go run ./tools/packet-audit fname-doc --check
```

`MULTI_CHAT` sb × `gms_v95` → **`verified`**. All exit 0.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/chat/serverbound/multi_test.go docs/packets/evidence/gms_v95/chat.serverbound.ChatMulti.yaml docs/packets/gates.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(atlas-packet): verify MULTI_CHAT serverbound on gms_v95"
```

---

## Task 14: W2 row 1 — verify `ENTER_CASHSHOP` serverbound (`CashShopEntry`)

Route: `packet:` (Task 3). Tier: **0** — the second and last of the deliberate
tier-0 evidence pins (§0/C2). `CWvsContext::SendMigrateToShopRequest` is absent
from the export and unresolved by name, so this needs the same
find-then-name-then-splice sequence as Task 13.

### Files

- `libs/atlas-packet/cash/serverbound/shop_entry_test.go` — add the v95 byte-golden test and marker (the file today is a 21-line round-trip with **no marker at all** — the only cash/serverbound codec without v95 coverage)
- `docs/packets/evidence/gms_v95/cash.serverbound.CashShopEntry.yaml` — **new file**
- `docs/packets/ida-exports/gms_v95.json` — **surgical splice of one entry only**, if Step 3 requires it
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Read-only: `libs/atlas-packet/cash/serverbound/shop_entry.go` — struct
`ShopEntry`, single field `updateTime uint32`.

Patterns to copy: `libs/atlas-packet/cash/serverbound/shop_entry_test.go` lines
1-21 (existing round-trip; keep it) and
`libs/atlas-packet/cash/serverbound/item_use_megaphone_test.go` for a sibling
byte-golden test carrying a `gms_v95` marker — every other codec in that
directory has one.

Module root: `libs/atlas-packet`.

- [ ] **Step 1: Locate and read the sender**

Database `ecc757f4`. Find the build site for opcode **43 = `0x2B`** by the
playbook §10 byte signature (`6A 2B 8D 8D ?? ?? ?? ?? E8` / `68 2B 00 00 00`),
scoping the search **binary-wide, not to an assumed region** — the teleport-rock
v48 case (playbook, "Is this cell n-a?" rule 1) is exactly the failure a narrow
scope produces. Name the function in the IDB if it is unnamed.

Compare the site to the `ENTER_MTS` precedent documented in
`tools/packet-audit/cmd/run.go` (`CWvsContext::SendMigrateToITCRequest` — a
**bodiless** request: ctor then `SendPacket()` with zero `Encode` calls). If
`ENTER_CASHSHOP` is likewise bodiless on v95, then `ShopEntry`'s `updateTime`
field is a wire divergence — surface it (playbook §4: wire fix first, own
commit, own review) rather than writing a fixture around it.

- [ ] **Step 2: Write the byte-golden test**

`TestShopEntryByteOutputV95`, tenant `pt.Variants[3]`, single case.

- If the client reads `Encode4(updateTime)`: fixture `updateTime = 0x11223344`,
  expected `[]byte{0x44, 0x33, 0x22, 0x11}`, length 4.
- If the client is bodiless: expected an empty slice, length 0 — **and** the
  divergence from `shop_entry.go` is resolved first per Step 1.

Assert length as well as content. Marker with the address from Step 1:

```go
// packet-audit:verify packet=cash/serverbound/CashShopEntry version=gms_v95 ida=<0xaddr>
```

- [ ] **Step 3: Run, pin, splice**

```
cd libs/atlas-packet && go test ./cash/serverbound/ -run TestShopEntryByteOutputV95 -v
```

```
go run ./tools/packet-audit evidence pin --packet cash/serverbound/CashShopEntry --version gms_v95 --ida "CWvsContext::SendMigrateToShopRequest" --category TIER1-FIXTURE
```

Same "not in export" → surgical splice rule as Task 12 Step 4. Use whatever name
the function actually carries after Step 1; `--ida` must match the export key
exactly. Hand-add `verifies:`. Record in `category` that this is a tier-0 cell
pinning evidence deliberately, because no report exists and none can be
generated without a harvest.

- [ ] **Step 4: Regenerate and gate**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
```

`ENTER_CASHSHOP` sb × `gms_v95` → **`verified`**.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/shop_entry_test.go docs/packets/evidence/gms_v95/cash.serverbound.CashShopEntry.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(atlas-packet): verify ENTER_CASHSHOP serverbound on gms_v95"
```

---

## Task 15: W2 row 9 — verify `NPC_ACTION` clientbound (`NpcAction`)

The only row already linked correctly. Its cell reads
`incomplete / tier-1 without fixture; verdict 🔍` — a report exists
(`docs/packets/audits/gms_v95/NpcAction.json`, `IDAName: CNpc::OnMove`), the
function is in the export at `0x678060`, and the codec has **no** marker for
`gms_v83/v84/v87/v95/jms_v185`. Straight playbook §3–§8, no export splice.

### Files

- `libs/atlas-packet/npc/clientbound/action_test.go` — add the v95 byte-golden test and marker
- `docs/packets/evidence/gms_v95/npc.clientbound.NpcAction.yaml` — **new file**
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Read-only: `libs/atlas-packet/npc/clientbound/action.go` — struct `Action`
(line 17) and its constructors.

Patterns to copy: `libs/atlas-packet/npc/clientbound/action_v72_test.go` lines
1-39 — the smallest complete file (one golden-byte test, no round-trip loop);
its marker line at 26 is the format model.

Module root: `libs/atlas-packet`.

- [ ] **Step 1: Read the client**

Decompile `CNpc::OnMove` @`0x678060` on database `ecc757f4` and descend into its
helper reads. `tools/packet-audit/cmd/run.go` around line 3659 records the shape
as `Decode1(action)` + `Decode1(chatIdx)` after the movement path — confirm
live; the comment is derived.

**Scope decision:** `NPCAction` also carries the movement arm, which consumes
the `options.types` table written in Task 11. This task fixtures the
**animation-only** arm (the shape the three existing legacy tests cover). If the
movement arm's read order differs materially, fixture it as a second case in the
same test rather than deferring it — Task 11 has already supplied the table it
needs.

- [ ] **Step 2: Write the failing byte-golden test**

`TestNpcActionByteOutputV95` in `action_test.go`, tenant `pt.Variants[3]`,
copying `action_v72_test.go`'s single-golden shape. Build the model with the
same constructor the v72 test uses. Fill the case table with the exact field
values and their expected bytes, one row per encode, each citing the decompile
address from Step 1 — do not carry the v72 expectations across without
re-deriving them.

Marker:

```go
// packet-audit:verify packet=npc/clientbound/NpcAction version=gms_v95 ida=0x678060
```

- [ ] **Step 3: Run, pin, regenerate**

```
cd libs/atlas-packet && go test ./npc/clientbound/ -run TestNpcActionByteOutputV95 -v
```

```
go run ./tools/packet-audit evidence pin --packet npc/clientbound/NpcAction --version gms_v95 --ida "CNpc::OnMove" --category TIER1-FIXTURE
```

`CNpc::OnMove` **is** in the export, so this should succeed without a splice.
Hand-add `verifies:`.

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
go run ./tools/packet-audit fname-doc --check
```

`NPC_ACTION` cb × `gms_v95` → **`verified`**; the `tier-1 without fixture;
verdict 🔍` note must be gone.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/npc/clientbound/action_test.go docs/packets/evidence/gms_v95/npc.clientbound.NpcAction.yaml docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "test(atlas-packet): verify NPC_ACTION clientbound on gms_v95"
```

---

## Task 16: W4 — prove and record `n-a` for the three remaining symbols

`LoginAuth` is **not** in this task — it is Task 4, because the registry's
`opcode: 0 / fname: ""` seed is absence-of-evidence, not evidence-of-absence.

**Open Question 5 is answered "no": do not create matrix rows to hold an `n-a`
verdict.** The grader derives ⬜ from *absence* in the version registry
(`opregistry.go` `Applicability`), so a row invented to hold a verdict is a row
the tool must be told to ignore. And `docs/packets/feature-na-evidence.yaml` is
narrower than the PRD assumes: the n-a-consistency gate consults it only when a
cell is `n-a` **while a same-feature sibling is `verified`**, and none of these
three symbols belongs to any family in `docs/packets/feature-families.yaml`. No
entry there is required or appropriate.

### Files

- `docs/tasks/task-146-v95-packet-verification-batch/v95-na-proof.md` — **new file**; the positive-absence proof for all three
- `docs/packets/audits/gms_v95/_unimplemented.json` — add one `{packet, reason}` entry for `ServerLoad`, if and only if Step 3 proves absence
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/status.json` — regenerated

Patterns to copy: the `cash/serverbound/CashItemUsePetSkill` entry near the end
of `docs/packets/audits/gms_v95/_unimplemented.json` — the `{packet, reason}`
shape (no `case` key), as opposed to the `{fname, case, reason}` shape the other
entries use.

Read-only: `libs/atlas-packet/login/clientbound/server_load.go` (struct
`ServerLoad`, line 15) and
`libs/atlas-packet/login/clientbound/server_load_test.go` (21 lines, no marker).

- [ ] **Step 1: `CreateSecurityHandle`**

Reach today:
`services/atlas-configurations/seed-data/templates/template_jms_185_1.json` at
`0x1A` only. No `fname`, no Atlas struct, no matrix row, no `gms_v95` registry
op. Prove absence on v95 per the playbook's "Is this cell `n-a`?" — **anchor on
invariants, never on a failed name search**: the opcode-to-handler jump arm in
the v95 login dispatcher, the `StringPool` ids the security-creation UI would
read, and the jms send site's structural twin. Write the anchors and addresses
into `v95-na-proof.md`. Nothing to record in tooling — there is no row and none
should be created.

- [ ] **Step 2: `WorldSelectHandle`**

Reach today:
`services/atlas-configurations/seed-data/templates/template_gms_12_1.json` at
`0x03` only. There is no `gms_v12` registry file at all. Same
invariant-anchored proof; task-folder record only.

- [ ] **Step 3: `ServerLoad` (writer)**

`libs/atlas-packet/login/clientbound/server_load.go` exists but has no v95
report and therefore no sub-struct row. Prove whether the v95 client has a
server-load receive arm — anchor on the login dispatcher's arms and the world/
channel-select UI's load-gauge read, and run the **mandatory sibling
cross-check** (playbook rule 3) against `SERVERSTATUS` / the server-list family,
which *is* present on v95: if a sibling decodes the same state, keep looking.

Only if absence is positively proven, add to `_unimplemented.json`:

```json
{
  "packet": "login/clientbound/LoginServerLoad",
  "reason": "task-146 gms_v95 sweep: <the positive-absence anchors, addresses, and the sibling cross-check result>"
}
```

Confirm the `packet` id against `qualifiedWriterName("login", "ServerLoad")`
= `LoginServerLoad` before writing it. If absence is **not** proven, record that
in `v95-na-proof.md` and leave `_unimplemented.json` untouched — an unproven
`n-a` is worse than an open cell.

- [ ] **Step 4: Regenerate and gate**

```
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Exit 0. The n-a-consistency gate runs inside `matrix --check`. If it reports
`… is n-a but sibling … is verified`, that is the playbook telling you the
feature exists — go back to Step 3, do not silence the gate.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-146-v95-packet-verification-batch/v95-na-proof.md docs/packets/audits/gms_v95/_unimplemented.json docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "docs(packets): positive-absence proofs for three gms_v95 n-a symbols"
```

---

## Task 17: coverage manifest, full gate sweep, and context capture

### Files

- `docs/tasks/task-146-v95-packet-verification-batch/coverage-manifest.yaml` — **new file**
- `docs/tasks/task-146-v95-packet-verification-batch/context.md` — **new file** if absent (written at plan time); update with the as-built outcome

Patterns to copy:
`docs/tasks/task-139-pet-auto-pot-validation/coverage-manifest.yaml` (133 lines)
— same op × version declaration shape.

- [ ] **Step 1: Write `coverage-manifest.yaml`**

Declare **every** op × version whose coverage this branch changed. From the plan
that is, on `gms_v95`: `ENTER_CASHSHOP` sb, `MULTI_CHAT` sb, `CHANGE_MAP` sb,
`NPC_TALK` sb, `CHANGE_MAP_SPECIAL` sb, `STORAGE` sb, `NPC_TALK` cb,
`CHECK_SPW_RESULT` cb, `NPC_ACTION` cb, `AUTO_DISTRIBUTE_AP` sb (opcode
correction), `LOGIN_AUTH` cb (Task 4 outcome), and the sub-struct
`npc/clientbound/NpcAskSlideMenuConversationDetail`. Derive the list
mechanically from `git diff <base>..HEAD` over `libs/atlas-packet`,
`docs/packets/registry/`, `docs/packets/evidence/` and `status.json` rather than
transcribing this paragraph — the manifest must match what actually landed.

- [ ] **Step 2: Run the full gate sweep**

Each as its own command line:

```
cd tools/packet-audit && go test ./...
```
```
go run ./tools/packet-audit fname-doc --check
```
```
go run ./tools/packet-audit operations --check
```
```
go run ./tools/packet-audit dispatcher-lint
```
```
go run ./tools/packet-audit doc-freshness --check
```
```
go run ./tools/packet-audit gate-check --check
```
```
go run ./tools/packet-audit matrix --check
```
```
tools/verify.sh
```

**Flagless `tools/verify.sh` must exit 0** — `--quick`/`--no-docker` skip the
bake and `-race` and do not count. Quote the actual exit codes; a flagged or
partial run is never "verified".

- [ ] **Step 3: Confirm the acceptance criteria mechanically**

Read `docs/packets/audits/status.json` and assert, in one pass:

- the nine PRD §4.2 cells all read `verified` on `gms_v95`;
- **no** cell listed in PRD §4.4 changed state;
- no non-v95 template file appears in `git diff --name-only <base>..HEAD`;
- the twelve PRD §4.3 entries all carry a populated `options` block;
- the W1 entries are present with the opcodes this plan tables
  (`CharacterAutoDistributeApHandle` at `0x63`, not `0x62`).

Record any deviation — including a dropped `PartyInviteRejectHandle` (Task 6
Step 3) — explicitly in `context.md`. A deviation stated is fine; a deviation
absorbed silently is not.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-146-v95-packet-verification-batch/coverage-manifest.yaml docs/tasks/task-146-v95-packet-verification-batch/context.md
git commit -m "docs(task-146): coverage manifest and as-built context"
```

---

## Review gate (not a task — the controller runs this)

Before any PR, per CLAUDE.md and PRD §8:

- `plan-adherence-reviewer` — every task actually implemented.
- `packet-completeness-critic` — against this task's `coverage-manifest.yaml`;
  must report no CHANGED-BUT-UNCLAIMED and no CLAIMED-BUT-UNVERIFIED.
- `backend-guidelines-reviewer` — Task 5 and Task 11 touch Go.
- `task-reviewer` per unit as tasks land.

A green `verify.sh` cannot see a cross-service seam defect: the v95 template is
consumed by both `atlas-channel` and `atlas-login` at socket-handler resolution
time, so trace the eleven new routes into both services by hand.

## Follow-ups (out of scope, file separately)

- The same missing-`options` gap on `gms_v87` (16/145 handlers with options),
  `gms_v92` (13/89) and `jms_v185`.
- Backfilling IDA-derived `gms_v83/v84/v87/v92/jms_v185` columns into the six
  new handler-keyed dispatcher docs, which would put those templates' tables
  under the `operations --check` gate. Until then they stay hand-maintained and
  un-gated — a knowing narrowing (design §4.1), not an oversight.
- `docs/packets/registry/gms_v92.yaml` carries the identical `LOGIN_AUTH`
  `opcode: 0 / fname: ""` seed. Left alone.
- Extending `packet-audit` to schema-validate `messageType`,
  `failedReasonCodes` and `types`. This task protects `types` with
  test-fixture coupling (Task 11) and leaves the other two on review.
