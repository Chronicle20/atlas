# Starting a new packet-audit version pass

> Reusable playbook for auditing `libs/atlas-packet/` against a (new or existing)
> client version. It captures the FINAL, corrected mechanics after task-080:
> the enhanced analyzer (A1–A5), the curated accepted-exclusions registry, the
> verify-against-IDA discipline, and the exact `packet-audit` invocation (the old
> task plans 027–069 documented an invocation that omitted the now-required
> `-template` flag — use the commands below verbatim).

---

## 1. The four-version baseline & where things live

The audit baseline spans **four client versions**:

| Version slug | Region / major.minor | Template | IDA export |
|---|---|---|---|
| `gms_v83`  | GMS 83.1  | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`  | `docs/packets/ida-exports/gms_v83.json` |
| `gms_v87`  | GMS 87.1  | `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`  | `docs/packets/ida-exports/gms_v87.json` |
| `gms_v95`  | GMS 95.1  | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`  | `docs/packets/ida-exports/gms_v95.json` |
| `jms_v185` | JMS 185.1 | `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` | `docs/packets/ida-exports/gms_jms_185.json` |

Directory map:

- **IDBs (the IDA databases themselves)** live outside the repo — one IDB per
  version, driven through IDA-MCP (`mcp__ida-pro__*`). They are the live oracle for
  `decompile_function` / `disassemble_function` / `get_function_by_name`. Open the
  IDB for the version you are verifying.
- **IDA exports** (`docs/packets/ida-exports/*.json`) are the static, checked-in
  harvest of each IDB's `Encode`/`Decode` read-order. These feed the analyzer when
  you don't want a live-MCP run. One JSON per version (note the JMS file is named
  `gms_jms_185.json`).
- **Accepted-exclusions registry**: `docs/packets/ida-exports/_pending.md`. A
  registry, NOT a deferral ledger — every entry is a blessed permanent exclusion or
  a pointer to a surfaced follow-up. Zero actionable items.
- **Per-version audit output**: `docs/packets/audits/<version>/` — one `SUMMARY.md`
  verdict table plus a per-packet `.md` detail page each.
- **Cross-task ledger**: `docs/packets/audits/gms_v95/TOTAL.md` (the roll-up lives
  under the v95 dir for historical reasons; it covers all four versions).
- **The analyzer**: `tools/packet-audit/` (module
  `github.com/Chronicle20/atlas/tools/packet-audit`).

Adding a NEW version means: produce its template, harvest its IDA export JSON, then
run the audit pointing at both. The output slug is derived from the template's
region + major version (`<region-lower>_v<major>`), e.g. a GMS 100.1 template emits
`docs/packets/audits/gms_v100/`.

---

## 2. Running `packet-audit` (the EXACT corrected invocation)

`packet-audit` is a cobra CLI (`tools/packet-audit/cmd/root.go`). It **requires
three flags**: `--csv-clientbound`, `--csv-serverbound`, and **`--template`** (the
old plans omitted `--template`; the tool errors without it — see root.go's
"missing required flags" guard). Both `--template` and `--ida-source` take **FILE**
paths. `--output` is a parent directory; the tool appends `<region>_v<major>/`
derived from the template and writes `SUMMARY.md` + per-packet `.md` there.

Build, then run one command per version (from the worktree root):

```bash
cd tools/packet-audit && go build ./...
```

```bash
# GMS v83
cd tools/packet-audit && go run . \
  --csv-clientbound "../../docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound "../../docs/packets/MapleStory Ops - ServerBound.csv" \
  --atlas-packet    ../../libs/atlas-packet \
  --template        ../../services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  --ida-source      ../../docs/packets/ida-exports/gms_v83.json \
  --output          ../../docs/packets/audits

# GMS v87
cd tools/packet-audit && go run . \
  --csv-clientbound "../../docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound "../../docs/packets/MapleStory Ops - ServerBound.csv" \
  --atlas-packet    ../../libs/atlas-packet \
  --template        ../../services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
  --ida-source      ../../docs/packets/ida-exports/gms_v87.json \
  --output          ../../docs/packets/audits

# GMS v95
cd tools/packet-audit && go run . \
  --csv-clientbound "../../docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound "../../docs/packets/MapleStory Ops - ServerBound.csv" \
  --atlas-packet    ../../libs/atlas-packet \
  --template        ../../services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  --ida-source      ../../docs/packets/ida-exports/gms_v95.json \
  --output          ../../docs/packets/audits

# JMS v185
cd tools/packet-audit && go run . \
  --csv-clientbound "../../docs/packets/MapleStory Ops - ClientBound.csv" \
  --csv-serverbound "../../docs/packets/MapleStory Ops - ServerBound.csv" \
  --atlas-packet    ../../libs/atlas-packet \
  --template        ../../services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  --ida-source      ../../docs/packets/ida-exports/gms_jms_185.json \
  --output          ../../docs/packets/audits
```

Notes:
- `--ida-source` accepts either a **file path** (static export, as above) or the
  literal `mcp` to drive a live IDA-MCP session against the open IDB.
- `--atlas-packet` defaults to `libs/atlas-packet`; pass the relative path when
  running from inside `tools/packet-audit`.
- Run all four versions before reading verdicts — a wire change is judged across the
  whole baseline, not one version.
- To capture a before/after `❌`/`🔍` inventory (e.g. when changing the analyzer),
  diff: `grep -rE '\| (❌|🔍) \|' docs/packets/audits/*/SUMMARY.md | sort`.

---

## 3. How SUMMARY / TOTAL / `_pending` relate

| Artifact | Scope | Who writes it |
|---|---|---|
| `docs/packets/audits/<version>/SUMMARY.md` | One version's per-packet verdict table (✅ / ❌ / 🔍) + detail `.md` pages | **Auto-generated** by `packet-audit`. Never hand-edit. |
| `docs/packets/audits/gms_v95/TOTAL.md` | Cross-task, cross-version roll-up: contributing tasks, coverage matrix, the four-version verdict totals, the audit-tool limitations, the completeness statement | Hand-maintained per pass. |
| `docs/packets/ida-exports/_pending.md` | Accepted-exclusions registry: every residual `❌`/`🔍` classified into a blessed permanent-exclusion category with IDA evidence; plus pointers to surfaced follow-up tasks | Hand-curated. **Zero actionable items** by invariant. |

Each verdict row in a SUMMARY carries exactly one glyph, so
`✅ = total rows − ❌ − 🔍`. The TOTAL §2a roll-up is just the SUMMARYs' glyph
counts summed across the four versions.

---

## 4. Version-gating conventions

### 4.1 Two-version divergence → inline tenant-context gate

When a field/shape differs across **≤2** version boundaries, gate it inline on the
tenant context. The canonical spelling is:

```go
t := tenant.MustFromContext(ctx)
v95Plus := t.Region() == "GMS" && t.MajorVersion() >= N
```

Exemplars in the tree:
- `libs/atlas-packet/stat/clientbound/changed.go` — `t.Region() == "GMS" && t.MajorVersion() >= 95`
- `libs/atlas-packet/login/serverbound/request.go` — `t.Region() == "GMS"` region gate
- `libs/atlas-packet/chat/serverbound/multi.go` — `hasUpdateTime := t.Region() == "GMS" && t.MajorVersion() >= 95`

Hard cap: **no encoder/decoder may exceed two levels of nested `if` guards** (a
review/`awk` policy). Where a guard composes GMS-and-JMS presence
(`(t.Region()=="GMS" && t.MajorVersion()>83) || t.Region()=="JMS"`, see
`field/clientbound/set_field.go`), that counts as one guard.

### 4.2 >2-version divergence → region-dispatched body (design §3.2)

When a divergence spans more than two versions, do NOT stack a third nested guard.
Dispatch at the **top** of the encode/decode closure to a **per-region helper
method**, each of which may carry ≤2 of its own guards:

```go
func (m EffectWeather) Encode(l logrus.FieldLogger, ctx context.Context) func(w *response.Writer) {
    t := tenant.MustFromContext(ctx)
    return func(w *response.Writer) {
        if t.Region() == "JMS" {
            m.encodeJMS(w)
        } else {
            m.encodeGMS(w)
        }
    }
}

func (m EffectWeather) encodeGMS(w *response.Writer) { /* ≤2 guards */ }
func (m EffectWeather) encodeJMS(w *response.Writer) { /* ≤2 guards */ }
```

Exemplars: `field/clientbound/effect_weather.go` (B1.5) and the JMS cash-shop
serverbound bodies (B5.1). **The analyzer now descends into these same-receiver
helpers** (task-080 region-dispatch helper descent, `tools/packet-audit/internal/
atlaspacket/analyzer.go` same-receiver HELPER descent, §4.7), so region-dispatched
packets analyze correctly instead of reporting an empty top-level body.

---

## 5. The analyzer as the de-noising baseline (A1–A5)

task-080 added five enhancements to `tools/packet-audit` so a clean re-run reads a
trustworthy signal. Know these so you can tell an analyzer artifact from a real
finding:

| # | Enhancement | What it suppresses |
|---|---|---|
| **A1** | Width-equivalence (`internal/diff/diff.go`, `widthEquivalent`) | `WriteByteArray(N)`/`WriteLong`/`WriteInt16+WriteShort(0)` vs a same-width `DecodeBuf`/`EncodeBuffer` — byte-equal, label-different. |
| **A2** | Name-qualification (`cmd/run.go`, `candidatesFromFName`) | `locateAtlasFile` struct-name collisions (e.g. `ChannelChange` audited against a buddy file). |
| **A3** | Sub-struct / opaque descent (`internal/atlaspacket`) | Self-describing sub-structs are descended; only genuinely-opaque residue is flagged. |
| **A4** | Early-return modeling | Mutually-exclusive `if/else` and early-`return` guards no longer double-counted (verified; covered by A1–A3). |
| **A5** | Region-dispatch helper descent (`analyzer.go` same-receiver HELPER descent) | `m.encodeJMS(w)` / `m.encodeGMS(t,w)` dispatch — walks the helper body instead of seeing an empty top-level closure. |

**What the analyzer still cannot resolve (expected residue, not bugs):**
- **Export read-order truncation** — the IDA-export JSON ends before a real Atlas
  trailing field, so the analyzer emits phantom `atlas: extra` / `atlas: short`
  rows. The wire is correct; the export simply didn't capture the full read-order.
- **Genuinely-opaque IDA types** — a single `DecodeBuf`/`EncodeBuf` token, or a
  struct with no decomposable layout (mob body, AvatarLook, `model.Asset`,
  `GW_ItemSlotBase`). The analyzer stops at the register boundary.

Both are expected and enumerated in `_pending.md` — when you see them in a fresh
pass, that is residue, not work.

---

## 6. Telling expected residue from a NEW real finding

A new pass will reproduce most of the four-version residue. Every residual
`❌`/`🔍` should fall into one of **four accepted-exclusion buckets** (full
definitions + the per-packet evidence tables are in `_pending.md`):

1. **TRUNCATION** — export read-order ended before a real Atlas trailing field
   (`extra`/`short` phantom rows). Wire verified by byte test / prior per-struct ✅.
2. **OPAQUE** — register-boundary IDA type with no decomposable layout.
3. **VERSION-ABSENT** — the FName/mode/feature is absent from this version's client
   (KMS-only, GMS-only, JMS-only, BBS-absent-in-JMS, unwired template seed). No
   counterpart to audit.
4. **REPRESENTATION-EQUIVALENCE** — identical wire bytes, different field
   decomposition (`WriteLong`≡`EncodeBuffer(8)`, `WriteInt64`≡FILETIME `DecodeBuf(8)`,
   4×`WriteInt32`≡`DecodeBuffer(16)` RECT, etc.).

(`_pending.md` also tracks two dispatcher-artifact classes — OP/MODE-PREFIX and
LOOP/EXCLUSIVE-BRANCH — which the analyzer largely suppresses post-A1–A5 but which
survive on a few mask/mode-driven packets.)

**Procedure for a fresh `❌`/`🔍`:**
1. Cross-reference the packet against the `_pending.md` evidence tables. If it's
   already classified there → expected residue, no action.
2. If it is NOT in `_pending.md` and not covered by a bucket above, it is either a
   **real wire bug** (→ open a new task, fix with a byte test) or a **new analyzer
   artifact** (→ extend `tools/packet-audit` §4.7 with a fixture + test). Never
   silently re-defer it.

---

## 7. Verify-against-IDA discipline (the non-negotiable)

**No wire change ships on the analyzer verdict alone.** The byte-level test is the
oracle, the IDA decompilation is the evidence, and the analyzer ✅/❌ is only a
triage signal. Concretely:

- Every wire change gets a `*_test.go` beside the packet that asserts the exact byte
  slice for **each** version it targets (use the model's own Builder /
  `pt.CreateContext("GMS", 95, 1)` / `pt.Variants` = GMS v28/v83/v87/v95 + JMS v185;
  no `*_testhelpers.go`).
- Each fix records the IDA `FName@address` and the read-order it was verified
  against (e.g. `CCashShop::OnBuy@0x47eaa7 → Encode1 isMaplePoint, Encode4 dwOption,
  Encode4 nCommSN`) in its audit entry / the registry.

This discipline caught **false plan premises** in task-080:
- **B1.3 `nItemPos`** — the plan asserted `ActionStart`/`ActionComplete` were missing
  an `Encode4(nItemPos)`. IDA across all of v83/v87/v95/JMS185 showed no such field;
  the existing decode was byte-correct. **Premise disproven, no change.**
- **B1.2 gate boundary** — the chat `Multi` `update_time` gate boundary asserted by
  the plan was corrected against IDA to `GMS >= 95` (not the wider boundary the plan
  assumed), pinned by `TestRequestTrailerShape`-style per-version byte tests.

If the analyzer and a byte-level IDA trace disagree, the IDA trace wins.
