# Task-069 — Implementation Context

Companion to `plan.md`. Captures the file map, repo state, and decisions the plan depends on.

## Worktree

- Path: `.worktrees/task-069-misc-domain-packet-audit/`
- Branch: `task-069-misc-domain-packet-audit`
- Forked from: `main` @ `414d7c872`

## Inventory — misc-domain packet files (verified at plan time)

```
libs/atlas-packet/account/serverbound/accept_tos.go            (already audited — task-027)
libs/atlas-packet/account/serverbound/register_pin.go
libs/atlas-packet/account/serverbound/set_gender.go
libs/atlas-packet/channel/clientbound/change.go
libs/atlas-packet/channel/serverbound/channel_change.go
libs/atlas-packet/fame/clientbound/response.go                  (multi-struct: ReceiveResponse, GiveResponse, ErrorResponse)
libs/atlas-packet/fame/response_body.go                         (body — register in TypeRegistry)
libs/atlas-packet/fame/serverbound/change.go
libs/atlas-packet/merchant/clientbound/operation.go             (multi-struct: 7 employee-shop variants — see below)
libs/atlas-packet/merchant/operation_body.go                    (body — register in TypeRegistry)
libs/atlas-packet/merchant/serverbound/operation.go
libs/atlas-packet/quest/clientbound/script_progress.go
libs/atlas-packet/quest/serverbound/action.go
libs/atlas-packet/quest/serverbound/action_complete.go
libs/atlas-packet/quest/serverbound/action_restore_lost_item.go
libs/atlas-packet/quest/serverbound/action_script_end.go
libs/atlas-packet/quest/serverbound/action_script_start.go
libs/atlas-packet/quest/serverbound/action_start.go
libs/atlas-packet/socket/clientbound/hello.go
libs/atlas-packet/socket/clientbound/ping.go
libs/atlas-packet/socket/serverbound/channel_connect.go
libs/atlas-packet/socket/serverbound/pong.go
libs/atlas-packet/socket/serverbound/start_error.go
libs/atlas-packet/stat/clientbound/changed.go
libs/atlas-packet/tool/uint128.go                               (utility — NOT a packet; documented in TOTAL.md and _pending.md)
libs/atlas-packet/ui/clientbound/disable.go
libs/atlas-packet/ui/clientbound/lock.go
libs/atlas-packet/ui/clientbound/open.go
libs/atlas-packet/ui/ui_open_body.go                            (body — register in TypeRegistry)
```

**Totals:** 21 packet files (1 already audited: `accept_tos.go`) + 3 body files + 1 utility.
**New SUMMARY rows expected:** ~20 packets (AcceptTos already in SUMMARY). Multi-struct files (`fame/clientbound/response.go` with 3 structs and `merchant/clientbound/operation.go` with 7 structs) may emit more than one row depending on how the audit pipeline groups them by FName.

## Writer / Handle constants (verified from grep)

| File | Constant | String |
|---|---|---|
| `account/serverbound/accept_tos.go` | `AcceptTosHandle` | `"AcceptTosHandle"` |
| `account/serverbound/register_pin.go` | `RegisterPinHandle` | `"RegisterPinHandle"` |
| `account/serverbound/set_gender.go` | `SetGenderHandle` | `"SetGenderHandle"` |
| `channel/clientbound/change.go` | `ChannelChangeWriter` | `"ChannelChange"` |
| `channel/serverbound/channel_change.go` | `ChannelChangeRequestHandle` | `"ChannelChangeHandle"` |
| `fame/clientbound/response.go` | `FameResponseWriter` | `"FameResponse"` |
| `fame/serverbound/change.go` | `FameChangeHandle` | `"FameChangeHandle"` |
| `merchant/clientbound/operation.go` | `HiredMerchantOperationWriter` | `"HiredMerchantOperation"` |
| `merchant/serverbound/operation.go` | `HiredMerchantOperationHandle` | `"HiredMerchantOperationHandle"` |
| `quest/clientbound/script_progress.go` | `ScriptProgressWriter` | `"ScriptProgress"` |
| `quest/serverbound/action.go` | `QuestActionHandle` | `"QuestActionHandle"` |
| `socket/clientbound/hello.go` | `HelloWriter` | `"Hello"` |
| `socket/clientbound/ping.go` | `PingWriter` | `"Ping"` |
| `socket/serverbound/channel_connect.go` | `CharacterLoggedInHandle` | `"CharacterLoggedInHandle"` |
| `socket/serverbound/pong.go` | `PongHandle` | `"PongHandle"` |
| `socket/serverbound/start_error.go` | `StartErrorHandle` | `"StartErrorHandle"` |
| `stat/clientbound/changed.go` | `StatChangedWriter` | `"StatChanged"` |
| `ui/clientbound/disable.go` | `UiDisableWriter` | `"UiDisable"` |
| `ui/clientbound/lock.go` | `UiLockWriter` | `"UiLock"` |
| `ui/clientbound/open.go` | `UiOpenWriter` | `"UiOpen"` |

Quest serverbound files use inline string returns (`"ActionComplete"`, `"ActionScriptStart"`, etc.) rather than named constants. Read each `Operation()` body for the canonical writer/handler string at audit time.

## Merchant clientbound multi-struct breakdown

`libs/atlas-packet/merchant/clientbound/operation.go` defines 7 employee-shop variants, all returning `HiredMerchantOperationWriter`:

- `OpenShop`
- `ErrorSimple`
- `ShopSearch`
- `ShopRename`
- `RemoteShopWarp`
- `ConfirmManage`
- `FreeFormNotice`

Per design §8: this is the **employee-shop** subset only. The **hire-merchant** subset (mode bytes for player-driven shops) lives elsewhere and is task-067's responsibility. Audit-report header for `HiredMerchantOperation.md` must explicitly state the in-scope mode bytes and reference task-067 for the rest. If audit time reveals that some of the 7 structs above are actually hire-merchant variants, partition the report into employee-shop and hire-merchant sub-sections and cross-link to task-067's verdict.

## Baseline state at branch fork

Verified via `cat docs/packets/audits/gms_v95/SUMMARY.md` on `task-069-misc-domain-packet-audit`:

- **SUMMARY rows:** 28 (login domain, from task-027).
- **Verdict counts:** ✅ 27 / ⚠️ 0 / ❌ 1.
- **Audit dirs present:** `docs/packets/audits/gms_v83/`, `docs/packets/audits/gms_v95/`.
- **IDA-export files present:** `docs/packets/ida-exports/gms_v95.json`, `gms_v83.json`, `_pending.md`. **Not present yet:** `gms_v87.json`, `gms_jms_185.json`.
- **AcceptTos already audited:** `AcceptTos.md` exists with verdict ✅. The audit is from task-027 (account/serverbound/accept_tos is the only misc-domain packet that overlapped with login-domain handler surface).
- **Sibling-task state in main:** tasks 028, 065, 066, 067, 068 are all in `.worktrees/` and NOT merged to main as of `414d7c872`. The TOTAL.md draft consequently pulls verdict counts from sibling-task branches via `git show <sibling-branch>:docs/packets/audits/gms_v95/SUMMARY.md` rather than from main.

Implications for the plan:
- The regression baseline diff is against the 28-row login SUMMARY, not 80+ rows.
- New IDA-export files (`gms_v87.json`, `gms_jms_185.json`) are created during Phase 3.
- TOTAL.md draft (Phase 4) pulls SUMMARY rows from sibling branches; the final pre-PR revision pulls from main once sibling tasks merge.

## Audit pipeline command

Same across all phases (template + ida-source swap per version):

```
go run ./tools/packet-audit \
  --csv-clientbound  "docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound  "docs/packets/MapleStory Ops - ServerBound.csv" \
  --template         services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --atlas-packet     libs/atlas-packet \
  --ida-source       docs/packets/ida-exports/gms_v95.json \
  --output           docs/packets/audits
```

Always run from the worktree root with relative paths — gitleaks check at Phase 4 grep's `/home/`.

> **EXECUTION CORRECTION (task-069, Phase 0 finding).** The plan text and the
> original version of this section passed `--output docs/packets/audits/gms_v95`
> (and `.../gms_v83`, etc. in Phase 3). That is WRONG: the tool itself appends
> `<region>_v<major>` to `--output` at `tools/packet-audit/cmd/run.go:42`
> (`filepath.Join(opts.Output, fmt.Sprintf("%s_v%d", lower(region), major))`).
> Passing the version subdir produces a NESTED `docs/packets/audits/gms_v95/gms_v95/`
> directory. **The correct value is always `--output docs/packets/audits`** — the
> parent — for EVERY version (v95, v83, v87, JMS185). The tool writes into
> `docs/packets/audits/{gms_v95,gms_v83,gms_v87,jms_v185}/` automatically.
> Treat every `--output docs/packets/audits/<version>` in plan.md as
> `--output docs/packets/audits`.

> **EXECUTION CORRECTION (task-069, Phase 0 finding) — regression gate is SEMANTIC,
> not byte-identical.** The plan's Phase 0 / Phase 1 regression gate asks for a
> byte-identical `diff` of SUMMARY.md against a snapshot. That is unachievable for
> two reasons: (1) SUMMARY row order is non-deterministic — it is built by ranging
> over Go maps (`tools/packet-audit/cmd/run.go:235` and `:243`), so order varies
> run-to-run; (2) the login-domain reports inherited at branch-fork used a stale
> `../../libs/atlas-packet/...` path prefix, whereas the current tool (and sibling
> audit tasks 028/065–068, which write `libs/atlas-packet/...`) emit a bare path.
> Running the audit with the corrected command normalizes the 28 login reports into
> the sibling convention (committed once in Phase 0). **The regression gate is
> therefore: the SORTED set of `packet → verdict` pairs must be unchanged** (no
> login verdict may flip; the baseline is 27 ✅ / 1 ❌, `CharacterList` being the
> sole ❌). Use this order-and-format-independent comparison wherever the plan says
> "diff against /tmp/summary-pre-task069.md":
>
> ```
> extract() { grep -oE '\[[A-Za-z]+\]\([A-Za-z]+\.md\) \| (✅|❌|⚠️)' "$1" | sed -E 's/\]\([^)]*\)//' | sort; }
> diff <(extract /tmp/summary-pre-task069.md) <(extract docs/packets/audits/gms_v95/SUMMARY.md)
> ```
>
> Empty diff = gate passes. The snapshot `/tmp/summary-pre-task069.md` should be
> taken from the Phase-0 *normalized* SUMMARY (post-reformat) so later phases
> compare like-for-like.

## Test patterns

The misc-domain encoders pair `Encode()` with `Decode()` on the same type. Round-trip tests use `libs/atlas-packet/test/{context.go,roundtrip.go}`:

```go
import pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"

func TestHelloRoundTrip(t *testing.T) {
    for _, v := range pt.Variants {
        t.Run(v.Name, func(t *testing.T) {
            ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
            input := NewHello(v.MajorVersion, v.MinorVersion, []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, 0x08)
            output := Hello{}
            pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
            // assert field-by-field
        })
    }
}
```

`pt.Variants` is `[]TenantVariant{GMS v28, GMS v83, GMS v95, JMS v185}`. The 4-variant test sweep required by PRD §4.4 is one `t.Run` per variant inside a single test function.

For client-bound packets where Atlas only encodes (no Decode pair), use the Encode-byte-equality form documented in `task-068` plan Task 6 step 5.

## TypeRegistry candidates (per design §5)

Phase 1 high-confidence (register up-front, with `registry_test.go` fixtures):

1. `fame/response_body.go` — consumed by `fame/clientbound/response.go`.
2. `ui/ui_open_body.go` — consumed by `ui/clientbound/open.go`.
3. `merchant/operation_body.go` — consumed by both directions of `merchant/operation.go`.

Phase 2 medium/low-confidence (register only if analyzer surfaces unresolved type):

- Quest reward sub-struct in `quest/serverbound/action_complete.go` / `action_start.go` — verify task-014/015 commit history before adding.
- Socket version-info block (IV pair + version pair) in `socket/clientbound/hello.go` — likely inline; verify.
- Channel migrate address block (host:port) in `channel/clientbound/change.go` — likely inline; verify.

## Cross-task coordination

- **task-014, task-015, task-023** — quest reward / quest start / quest skill-gate. All merged to main. Before any quest file fix, run `git log --oneline -- libs/atlas-packet/quest/` and read commits from these tasks. Treat existing `Region/MajorVersion` gates as load-bearing; don't widen or narrow without IDA evidence from the same version context the prior task used.
- **task-027 (merged)** — established the pipeline; `accept_tos` already audited as part of login-domain pass.
- **task-028, task-065, task-066, task-067, task-068** — in flight in sibling worktrees. Their SUMMARY rows are inputs to TOTAL.md. If any are still in flight at Phase 4, TOTAL.md uses `(draft)` annotations per design §10.4 and a pre-PR sweep updates them.
- **task-067 (commerce)** — owns hire-merchant `interaction/` packets. The employee-shop variants in `merchant/clientbound/operation.go` are task-069's scope; cross-link in the audit report header.

## Key design decisions inherited

- **No analyzer changes.** If audit panics or surfaces a new cycle, STOP and escalate — do not inline-fix `tools/packet-audit/internal/atlaspacket/analyzer.go`.
- **2-deep nesting cap.** No misc-domain encoder gets a 3-deep exception (`set_field.go`'s carve-out was world-domain only). 3+ → STOP, defer to `_pending.md`.
- **Flat audit-report layout.** `docs/packets/audits/gms_v95/<FName>.{md,json}` interleaves with prior-domain reports. No per-domain subdirectories.
- **Bare handlers defer.** No descent into `services/atlas-account/`, `services/atlas-channel/`, `services/atlas-quest/`. Handlers with no `libs/atlas-packet` decoder get a `_pending.md` row.
- **Ack footer policy.** Ack footer is the LAST line written to each report. If a re-run is needed: `git checkout HEAD -- docs/packets/audits/gms_v95/<report>.md` first.
- **gitleaks.** Absolute paths under `/home/` must not appear in any audit report. Pre-PR grep is mandatory.

## Reference points

- `docs/tasks/task-027-atlas-packet-v95-audit/` (on main) — pipeline origin + closing-memo template.
- `.worktrees/task-028-character-domain-audit/docs/tasks/task-028-character-domain-audit/` — most-recent pipeline-scaling pattern; `EncodeForeign` registry, cycle guard, suffix-taint walker, ack-footer convention.
- `.worktrees/task-068-world-domain-packet-audit/docs/tasks/task-068-world-domain-packet-audit/plan.md` — structural template for this plan; phase / sub-phase / task layout.
- `libs/atlas-packet/test/{context.go,roundtrip.go}` — `pt.Variants` and `pt.RoundTrip` helpers.
- `tools/packet-audit/internal/atlaspacket/registry.go` (+ `registry_test.go`) — TypeRegistry additions land here.
- `services/atlas-configurations/seed-data/templates/template_gms_{12,28,83,87,92,95}_1.json` and `template_jms_185_1.json` — opcode/enum sites.
- `docs/packets/MapleStory Ops - ClientBound.csv`, `docs/packets/MapleStory Ops - ServerBound.csv` — opcode↔writer/handler mapping used by the audit pipeline.
