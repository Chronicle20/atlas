# Packet process — index & source of truth

Top-of-tree index for all packet-audit work. Start here, then follow the one
canonical playbook for your task type. This file is also the **machine-lintable
source of truth**: the fenced `packet-process-facts` block below states the facts
every playbook asserts (version set, baseline status, CI gates), so a freshness
lint (task-169 FR-2.3, planned) can diff the docs against the tool.

Do not restate these facts in prose elsewhere and let them drift — link here.

## Task type → entry point → canonical playbook

Each task type has ONE canonical playbook (the procedure) and an executable
entry point (a slash command / agent that drives it).

| Task type | Entry point | Canonical playbook |
|---|---|---|
| Implement a new feature codec (clientbound or serverbound) | `/implement-packet` command + `packet-implementer` agent — **exists** | [`IMPLEMENTING_A_PACKET.md`](IMPLEMENTING_A_PACKET.md) |
| Bring up a new client-version column | `/bringup-version` command — **exists** | [`audits/STARTING_A_NEW_VERSION_PASS.md`](audits/STARTING_A_NEW_VERSION_PASS.md) |
| Audit / implement a mode-prefix dispatcher family | `family-auditor` agent (read-only audit) — **exists** · `dispatcher-family-implementer` agent (do-mode) — **exists** | [`DISPATCHER_FAMILY.md`](DISPATCHER_FAMILY.md) |

Every task type's leaf step — promoting one packet × version matrix cell to
`✅` — is the single-cell verify procedure, shared by all of the above:

| Leaf step | Entry point | Canonical playbook |
|---|---|---|
| Verify one packet × version cell | `/verify-packet` command + `packet-verifier` agent — **exists** | [`audits/VERIFYING_A_PACKET.md`](audits/VERIFYING_A_PACKET.md) |

## Version set

The coverage matrix tracks **9** client versions, in this column order (source:
`matrix.VersionKeys` in `tools/packet-audit/internal/matrix/model.go`):

`gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`,
`gms_v95`, `jms_v185`.

Each has a registry (`docs/packets/registry/<key>.yaml`), an IDA export
(`docs/packets/ida-exports/<key>.json`; `jms_v185` uses `gms_jms_185.json`), a
seed template, and an audit dir (`docs/packets/audits/<key>/`).

## Baseline status

Both dispatcher baselines are **empty — every family has graduated** to the
canonical discrete-per-mode pattern:

- `docs/packets/dispatcher-lint-baseline.yaml` → `exempt_families: []`
  (guild/message/party/buddy all migrated; the list only shrinks — task-105).
- `docs/packets/evidence/families.yaml` → the `dispatchers:` list is entirely
  commented (all cash/mts/shop/storage/messenger/interaction/trunk arms
  graduated — task-096). Because it is empty, the `🧩` family state currently
  caps **no** op; a newly-added family would need an explicit entry to cap.

## CI gates (`.github/workflows/packet-matrix.yml`)

All of these run on every PR touching `tools/packet-audit/**`,
`libs/atlas-packet/**`, `docs/packets/**`, or the seed templates. Each is
blocking (no `continue-on-error`):

1. **Packet-audit tests** — `cd tools/packet-audit && go test ./...`
2. **fname-doc check** — `packet-audit fname-doc --check`
3. **operations table check** — `packet-audit operations --check`
4. **dispatcher lint** — `packet-audit dispatcher-lint` (also enforces the
   family-cap guard, task-169 FR-5.1).
5. **doc-freshness check** — `packet-audit doc-freshness --check` (asserts this
   facts block still matches the tool's ground truth; task-169 FR-2.3).
6. **gate-check** — `packet-audit gate-check --check` (asserts every version-gated
   wire divergence in [`gates.yaml`](gates.yaml) has a verified byte-fixture on
   BOTH adjacent straddling versions; task-169 FR-3.1b).
7. **Coverage matrix check** — `packet-audit matrix --check` (hard gate: ANY
   non-zero exit fails CI — a `🟥` conflict, a stale committed
   STATUS.md/status.json, a fatal finding, or a runtime error). The
   registry-seed conflict backlog was burned to zero (task-085), so a clean
   tree exits 0; there is no grandfathering.

The `gate-lint` idiom check (task-169 FR-3.1a) is intentionally **not** a
blocking CI gate. Task-169 T4.1b narrowed it to the genuinely off-by-one-prone
forms only — strict `MajorVersion() > N` and inclusive `<= N` (and their
left-operand twins) — dropping the ~185 correct `>= N` / `< N` idiom hits Phase
4a flagged. The narrowed form still hits **35** sites, but every one is a
task-113 code-gate-audit VERIFIED-CORRECT gate whose boundary sits between two
adjacent version columns (e.g. `>87` == `>=95` today); going blocking would
demand an allow-annotation on each of those 35 wire-source files, so it stays
report-only (`packet-audit gate-lint`). The export non-destructive-overwrite
guard (task-169 FR-3.2) is a runtime behavior of `packet-audit export`, not a CI
check (CI never harvests).

## Machine-checkable facts

The freshness lint (task-169 FR-2.3, planned) parses this block and asserts it
against the tool. Keep it in sync with the sources cited above; do not hand-edit
to disagree with them.

```yaml
# packet-process-facts
version_count: 9
version_keys:
  - gms_v48
  - gms_v61
  - gms_v72
  - gms_v79
  - gms_v83
  - gms_v84
  - gms_v87
  - gms_v95
  - jms_v185
dispatcher_lint_baseline_families: []   # docs/packets/dispatcher-lint-baseline.yaml
family_cap_dispatchers: []              # docs/packets/evidence/families.yaml (all commented/graduated)
ci_gates:
  - packet-audit-tests          # cd tools/packet-audit && go test ./...
  - fname-doc-check             # packet-audit fname-doc --check
  - operations-check            # packet-audit operations --check
  - dispatcher-lint             # packet-audit dispatcher-lint (incl. family-cap)
  - doc-freshness-check         # packet-audit doc-freshness --check
  - gate-check                  # packet-audit gate-check --check (boundary-fixture pairs)
  - matrix-check                # packet-audit matrix --check (hard gate)
matrix_check_hard_gate: true
```

## Coverage manifest (packet tasks)

A packet task (new codec, version bring-up, dispatcher family) declares its
intended scope UP FRONT in `docs/tasks/<task>/coverage-manifest.yaml`. The
`packet-completeness-critic` agent (run in the pre-PR review step) diffs this
manifest against the branch's actual git + matrix delta and flags the two
failure modes of the class-8 "semantic scope hole":

- **CHANGED-BUT-UNCLAIMED** — a codec struct or version gate moved in the diff
  but the packet isn't in `ops` (and isn't in `out_of_scope`). This is the scope
  hole: work landed that the task never declared and no one verified.
- **CLAIMED-BUT-UNVERIFIED** — a manifest `op × version` has no `verified` cell
  in the final `status.json`. The task promised coverage it didn't deliver.

Schema (`docs/tasks/<task>/coverage-manifest.yaml`):

```yaml
# coverage-manifest
ops:                 # packets this task adds/changes coverage for.
  - CHARACTER_SPAWN                        # an op name (status.json `op`), OR
  - character/clientbound/CharacterSpawn   # a packet path (status.json `packet`)
versions:            # version keys the task targets (subset of the 9).
  - gms_v83
  - gms_v84
fields:              # OPTIONAL free-text notes of the specific gated fields touched.
  - "character/clientbound/CharacterSpawn: v84 DR-block"
out_of_scope:        # packets the diff may touch that are DELIBERATELY not this
  - model/asset      # task's coverage (incidental edit, shared-struct churn).
                     # Listed here so the critic won't flag them CHANGED-BUT-UNCLAIMED.
```

`ops` entries accept either the status.json `op` name or the `packet` path; the
critic resolves both. Keep the manifest honest — an `out_of_scope` entry is a
claim that the touch is intentional and needs no verification, not a way to
silence the critic.

## Matrix cell states

`✅` verified · `🧩` family (mode-prefix dispatcher; sub-arms unverified) ·
`🟡` partial · `❌` incomplete · `⬜` n-a · `🟥` conflict. (Source:
`State.Symbol()` / the render legend in `tools/packet-audit/internal/matrix/`.)
