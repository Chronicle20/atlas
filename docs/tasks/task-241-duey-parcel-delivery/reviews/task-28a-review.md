# Review: Task 28a — route DUEY_ACTION serverbound handler and enumerate parcel_pending

Unit reviewed: commit `ee2267d64` (range `283a618da..ee2267d64`).
Brief: `.superpowers/sdd/plan/task-28a-brief.md`
Implementer report: `.superpowers/sdd/plan/task-28a-report.md`

## Scope confirmed

`git show --stat ee2267d64` shows exactly the 10 files the brief and report
claim: `docs/packets/dispatchers/cash_shop_operation.yaml`,
`docs/packets/dispatchers/duey_action.yaml`, and the eight in-span templates
(`template_gms_{72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`).
No Go file, no `STATUS.md`/`status.json`, no ledger/review-artifact sweep-in.
Matches the stated scope.

## 1. Routing defect fixed end to end — PASS

`docs/packets/dispatchers/duey_action.yaml:52-63` (post-commit): header key
changed from `writer: DueyAction` to `handler: DueyActionHandle`, matching
the Go constant `libs/atlas-packet/parcel/serverbound/action.go:13` verbatim,
and registered at `services/atlas-channel/atlas.com/channel/main.go:1011`
(`handlerMap[parcelsb.DueyActionHandle] = handler.DueyActionHandleFunc`).
An `opcodes:` block was added with all eight in-span versions.

Verified by extracting the rendered `socket/handlers` entry from each of the
eight templates (`json.load`, not grep):

| version | opCode found | opCode expected (brief) | match | operations keys |
|---|---|---|---|---|
| gms_72 | 0x040 | 0x040 | yes | SEND=2 RECEIVE=4 DISCARD=5 CLOSE=7 |
| gms_79 | 0x03F | 0x03F | yes | SEND=2 RECEIVE=4 DISCARD=5 CLOSE=7 |
| gms_83 | 0x041 | 0x041 | yes | SEND=2 RECEIVE=4 DISCARD=5 CLOSE=7 |
| gms_84 | 0x041 | 0x041 | yes | SEND=2 RECEIVE=4 DISCARD=5 CLOSE=7 |
| gms_87 | 0x044 | 0x044 | yes | SEND=2 RECEIVE=4 DISCARD=5 CLOSE=7 |
| gms_92 | 0x047 | 0x047 | yes | SEND=2 RECEIVE=4 DISCARD=5 CLOSE=7 |
| gms_95 | 0x046 | 0x046 | yes | SEND=2 RECEIVE=4 DISCARD=5 CLOSE=7 |
| jms_185 | 0x039 | 0x039 | yes | SEND=3 RECEIVE=5 DISCARD=6 CLOSE=8 |

jms_185's +1 shift (SEND 3/RECEIVE 5/DISCARD 6/CLOSE 8 vs GMS's
2/4/5/7) is present exactly as the brief and `duey_action.yaml`'s
`operations:` table specify — this is the specific failure mode the review
task called out and it is correct.

Negative controls: `template_gms_12_1.json`, `template_gms_48_1.json`,
`template_gms_61_1.json` all have zero `DueyActionHandle` entries, as
expected (op is `n-a` at those versions).

## 2. Regenerated, not hand-edited — PASS

Ran `go run ./tools/packet-audit operations` myself on the checked-out
commit: `operations: wrote 0 template(s)`, exit 0, and
`git status --short` over `docs/packets/` and the templates directory shows
no diff. The committed templates are exactly what the generator produces
from the YAML — Constraint 2 satisfied.

All four gates, run individually and independently of the implementer's
report:

```
go run ./tools/packet-audit operations --check   → "operations check OK (0 absent-writer note(s))"   exit 0
go run ./tools/packet-audit matrix --check        → two pre-existing n-a-evidence notes only           exit 0
go run ./tools/packet-audit dispatcher-lint       → "dispatcher-lint: clean"                            exit 0
go run ./tools/packet-audit fname-doc --check     → "fname-doc check OK (294 structs ... carry no fname)" exit 0
```

Matches the implementer report's transcript verbatim.

## 3. Opcode correctness — PASS

All eight opcodes (`docs/packets/dispatchers/duey_action.yaml:52`) match the
brief's pinned table exactly: gms_v72=0x040, gms_v79=0x03F, gms_v83=0x041,
gms_v84=0x041, gms_v87=0x044, gms_v92=0x047, gms_v95=0x046,
jms_v185=0x039. gms_v48/gms_v61 got no entry, confirmed by the negative
control above (no `DueyActionHandle` in `template_gms_48_1.json` /
`template_gms_61_1.json`).

Single-sourcing of v72/v79/v92 is disclosed, not silently presented as
double-confirmed. `docs/packets/dispatchers/duey_action.yaml:13-16` states:
"gms_v72, gms_v79 and gms_v92 have only that one status.json source (the
v72/v79 support docs say n-a with no opcode, and there is no gms_v92.md)."
This is an honest disclosure in the YAML comment header, matching the
brief's requirement and the controller's pre-accepted ruling. Acceptable per
task framing — not re-litigated.

## 4. parcel_pending enumeration entry — PASS

`docs/packets/dispatchers/cash_shop_operation.yaml:478-479` adds:
```yaml
  - key: parcel_pending
    alias_of: CANNOT_TRANSFER_OUT
```
immediately after `mts_listings_open` (line 477), in the same alias form.
Ran `go run ./tools/packet-audit operations --check` myself and grepped for
`EXTRA` in the output — none present. The nine `EXTRA` violations
(`parcel_pending` key in template but not in enumeration, one per GMS
version) reported in the brief are gone.

## 5. Scope — PASS

- `git show --stat ee2267d64` contains zero `.go` files — Constraint 3
  satisfied (no Go change).
- `git diff 283a618da..ee2267d64 -- docs/packets/audits/STATUS.md
  docs/packets/audits/status.json` is empty — neither file changed in this
  commit, consistent with the report's claim that the matrix did not move
  and no regeneration was needed. Constraint 1 satisfied (no hand edit,
  and in fact no edit at all).
- `git status --short docs/tasks/task-241-duey-parcel-delivery/` still
  shows the pre-existing dirty `agent-ledger.tsv` modification and the
  four untracked `reviews/task-2{4,5,6,7}-review.md` files, all left
  uncommitted — the commit did not sweep these in.

## Not evaluable

None. The full brief surface (both YAML edits, all eight regenerated
templates, all four gates, opcode table, scope constraints) was directly
verifiable within this commit and its stated dependencies (Go constant
definitions, the character_interaction_handle.yaml convention file).

## Verdict

APPROVED. Every brief requirement is satisfied and independently
reproduced: the routing defect is fixed end to end (config now reaches
`duey_action.go:31-60`'s `isDueyAction` config-resolved comparison), the
templates are a byte-for-byte generator artifact (verified by a live
re-run, not trusted from the report), all four gates exit 0, the
enumeration gap is closed, and scope stayed within the two YAMLs plus the
eight regenerated templates with no Go change and no incidental sweep-in.
