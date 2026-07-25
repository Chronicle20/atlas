# Re-auditing an existing version column

Maintenance playbook for re-auditing a version column that is **already brought
up** (its registry, template, export, and audit dir all exist) after something
drifts. This is the counterpart to
[`audits/STARTING_A_NEW_VERSION_PASS.md`](audits/STARTING_A_NEW_VERSION_PASS.md):
that one stands a column up from nothing; this one re-checks a standing column
without re-harvesting the world.

It reuses the same diagnostic toolkit `STARTING` §1.4 documents for bring-up —
`validate` / `decompose` / `triage` / `diff-shape` / `infer` — but frames it
around the three maintenance triggers, so a maintainer doesn't have to read a
new-version pass to fix an existing one.

> **Read-only vs mutating.** Every tool below reads the committed baseline export
> and a live IDB and writes only a report/proposal — **none mutates a codec,
> registry, template, or the committed export**. The one exception in the family
> is `resolve-dispatch` (it writes selectors into the baseline); it is a bring-up
> tool, not a maintenance one, and is intentionally omitted here. To *change* a
> committed export use the surgical `export --splice` path
> ([`audits/VERIFYING_A_PACKET.md`](audits/VERIFYING_A_PACKET.md) §10) — never a
> full re-export overwrite.

All commands run from the worktree root and need a live IDA-MCP instance for the
target version; select it with `--ida-port <port>` (0 = default active instance).
Confirm the loaded IDB matches the version you are auditing before you read.

## The diagnostic toolkit

Each entry: one-line purpose, its flags (verified against
`tools/packet-audit/cmd/root.go`), and when to reach for it. Required flags are
marked; defaults are shown where the flag has one.

### `validate` — baseline vs live IDB

Cross-checks the hand-authored/committed baseline export against the open IDB and
reports `divergent` / `missing-mode` entries (dispatcher applicability included).

```
--version <key>        target version key (required)
--report <path>        output markdown report path (required)
--baseline <path>      baseline export JSON (default: docs/packets/ida-exports/<version>.json)
--allowlist <path>     unimplemented-case allowlist (default: docs/packets/audits/<auditdir>/_unimplemented.json)
--descent-depth <n>    max helper-descent recursion depth (default 6)
--ida-url <url>        IDA-MCP HTTP endpoint (default http://192.168.20.3:13337/mcp)
--ida-port <n>         IDA-MCP instance to select (default 0 = active)
--ida-timeout <dur>    per-call IDA-MCP timeout (default 1m0s)
```

Reach for it first on any re-audit: it is the cheapest "does the committed
baseline still match the binary?" signal. Triage its `divergent` rows with
`diff-shape`; allowlist genuine `missing-mode` cases into the version's
`_unimplemented.json`.

### `decompose` — extend the baseline with live reads

Walks every exported entry, pulls its live IDA read order, and writes an
**extended** baseline plus a report — the input `triage` compares against.

```
--version <key>        target version key (required)
--out <path>           output extended baseline JSON (required)
--report <path>        output markdown report path (required)
--baseline <path>      baseline export JSON (default: docs/packets/ida-exports/<version>.json)
--audit-dir <path>     committed audit dir (default: docs/packets/audits/<version>)
--descent-depth <n>    max helper-descent recursion depth (default 6)
--ida-url <url>        IDA-MCP HTTP endpoint (default http://192.168.20.3:13337/mcp)
--ida-port <n>         IDA-MCP instance to select (default 0 = active)
--ida-timeout <dur>    per-call IDA-MCP timeout (default 1m0s)
```

Reach for it when `validate` flags divergence broadly (a re-harvest trigger) and
you want the full live read-order set to diff, not just a pass/fail. It only
*upgrades* existing reports; it never generates new audit reports.

> **JMS quirk**: `--version gms_jms_185` defaults `--audit-dir` to a
> non-existent `docs/packets/audits/gms_jms_185`; pass `--audit-dir
> docs/packets/audits/jms_v185` explicitly.

### `triage` — divergence worklist

Produces a divergent-entry worklist (markdown) from the extended baseline: the
per-op list of "committed baseline says X, live IDB says Y" you work down.

```
--version <key>        target version key (required)
--report <path>        output markdown worklist path (required)
--baseline <path>      baseline export JSON (default: docs/packets/ida-exports/<version>.json)
--audit-dir <path>     committed audit dir (default: docs/packets/audits/<version>)
--descent-depth <n>    max helper-descent recursion depth (default 6)
--ida-url <url>        IDA-MCP HTTP endpoint (default http://192.168.20.3:13337/mcp)
--ida-port <n>         IDA-MCP instance to select (default 0 = active)
--ida-timeout <dur>    per-call IDA-MCP timeout (default 1m0s)
```

Reach for it after `decompose` to turn the extended baseline into an ordered fix
list. Same JMS `--audit-dir` quirk as `decompose`.

### `diff-shape` — read-only shape diagnostic

Read-only structural diff of one version's packet shapes (baseline vs live) — no
worklist, no mutation, just the shape delta for a suspected field-level drift.

```
--version <key>        target version key (required)
--report <path>        output markdown report path (required)
--baseline <path>      baseline export JSON (default: docs/packets/ida-exports/<version>.json)
--descent-depth <n>    max helper-descent recursion depth (default 6)
--ida-url <url>        IDA-MCP HTTP endpoint (default http://192.168.20.3:13337/mcp)
--ida-port <n>         IDA-MCP instance to select (default 0 = active)
--ida-timeout <dur>    per-call IDA-MCP timeout (default 1m0s)
```

Reach for it to confirm a single `validate` `divergent` row: is the read order
actually different (material) or just Hex-Rays cosmetic churn? This is the
arbiter for the hash-drift decision below.

### `infer` — propose selectors

Proposes dispatcher selectors (a proposal JSON, with a high-confidence roll-up) —
**read-only**: it never writes into the baseline (that is `resolve-dispatch`,
which is a bring-up tool and out of scope for maintenance).

```
--version <key>        target version key (required)
--out <path>           output proposal JSON path (required)
--baseline <path>      baseline export JSON (default: docs/packets/ida-exports/<version>.json)
--min-confidence <f>   high-confidence threshold for the roll-up (default 0.6)
--descent-depth <n>    max helper-descent recursion depth (default 6)
--ida-url <url>        IDA-MCP HTTP endpoint (default http://192.168.20.3:13337/mcp)
--ida-port <n>         IDA-MCP instance to select (default 0 = active)
--ida-timeout <dur>    per-call IDA-MCP timeout (default 1m0s)
```

Reach for it when a re-audit implicates a mode-prefix dispatcher family and you
want a *proposal* for which selector each arm reads, without touching the
committed baseline. Feed a `family-auditor` report or a `triage` worklist into
deciding which arms to inspect.

## The three triggers

### 1. A family-audit bug

The `family-auditor` agent (read-only) reported a per-arm gap or an
operations-table mismatch for a dispatcher family
([`DISPATCHER_FAMILY.md`](DISPATCHER_FAMILY.md)). To confirm the fix scope on the
affected version(s):

1. `validate` the version → is the family's op flagged `divergent` / `missing-mode`?
2. `infer` the family's selectors → does the proposal match the yaml `modes[version]`?
3. Take the confirmed gap to the do-mode fix: a `dispatcher-family-implementer`
   pass (arm bodies) or a targeted `packet-verifier` fan-out (unverified arms).

The audit reports the gap; this playbook confirms it against the live IDB before
any codec changes.

### 2. An export re-harvest

The committed export drifted or is stale (new fnames needed, a bad harvest, a
re-export request). **Do not overwrite the committed export** — a full re-export
drifts ~150 unrelated function keys (`VERIFYING_A_PACKET.md` §10). Instead:

1. `decompose` → live extended baseline for the whole version.
2. `triage` → the divergent-entry worklist (which entries actually changed).
3. For each entry that genuinely changed, surgically merge it with
   `export --splice <fname>` (single-entry merge, all other entries preserved) —
   see `VERIFYING_A_PACKET.md` §10. `--splice` requires the existing `--output`
   and merges only the named FName.
4. Regenerate the matrix and re-check any cell whose evidence hash the splice
   moved (trigger 3).

### 3. A matrix cell that degraded

A previously-`✅` cell went `❌` / `🟡` (`matrix --check` emits `stale` / `drift`
/ `orphan`). The remediation branches on *why* — the canonical decision tree is
[`audits/STARTING_A_NEW_VERSION_PASS.md`](audits/STARTING_A_NEW_VERSION_PASS.md)
§5.2 (degraded verified cells). This playbook adds the diagnostic step:

- **Evidence hash drift** (re-export changed the decompile text). Use `diff-shape`
  (or `triage`) to decide material vs cosmetic:
  - **Cosmetic** (Hex-Rays variable/label renaming, read order unchanged) →
    re-pin, do **not** re-verify:

    ```bash
    go run ./tools/packet-audit evidence pin \
      --packet  <pkg/dir/Struct> \
      --version <version-key> \
      --ida     "<FName as it appears in the export>" \
      --category <TIER1-FIXTURE|OPAQUE|TRUNCATION|...>
    ```

  - **Material** (the read order actually differs) → full re-verification via
    `VERIFYING_A_PACKET.md` §3–8. A changed read order is a finding, not a re-pin.
- **Broken test linkage / tool verdict flip** → follow `STARTING` §5.2 legs 2–3;
  use `triage` / `decompose` to re-run the live decompile and hand-confirm which
  side is right (the IDA trace wins over any analyzer verdict).

After any re-pin or splice, regenerate and re-check:

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```

Commit the re-pinned evidence / spliced export together with the regenerated
`STATUS.md` / `status.json`. Never hand-edit `STATUS.md`.

## Cross-references

- [`PROCESS.md`](PROCESS.md) — top-of-tree index; version set, baseline status, CI gates.
- [`audits/STARTING_A_NEW_VERSION_PASS.md`](audits/STARTING_A_NEW_VERSION_PASS.md)
  — bring-up counterpart; §1.4 (toolkit in bring-up context), §5 (degradation remediation).
- [`audits/VERIFYING_A_PACKET.md`](audits/VERIFYING_A_PACKET.md) — §7 evidence pin,
  §9 serverbound 3-artifact rule, §10 export hygiene / `--splice`.
- [`DISPATCHER_FAMILY.md`](DISPATCHER_FAMILY.md) — the family pattern the
  `family-auditor` trigger audits against.
