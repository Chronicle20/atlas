# Player-cast mists II Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver Shadower Smokescreen (4221006), Blaze Wizard Flame Gear (12111005), Night Walker Poison Bomb (14111006), and Evan Recovery Aura (22161003) as player-cast mists on task-200's mechanism, including the Evan wire→Identity binding prerequisite that currently makes Recovery Aura undispatchable.

**Architecture:** atlas-maps owns the mist contract, registry, and tick; it gains two `EffectKind` values (`PROTECTION`, `RECOVERY`), a `RecoveryMp`/`PartyMemberIds` magnitude pair, an effect-kind-aware `nType` derivation, and a `tickRecovery` arm that emits `COMMAND_TOPIC_CHARACTER` `CHANGE_MP`. atlas-channel gains a shared `mistcast` cast helper (the four new handlers plus a refactored `poisonmist` differ only in a `Params` struct), a full mirror of the mist contract guarded by a new mirror guard, and — for Smokescreen — a channel-local protection-mist registry populated from `EVENT_TOPIC_MIST` and consulted as a short-circuit at the top of `processDamageTaken`, matching the client's own `IsSmokeAreaByPoint` short-circuit.

**Tech Stack:** Go 1.24, Kafka (`segmentio/kafka-go`) via `libs/atlas-kafka`, JSON:API REST via `libs/atlas-rest`, `logrus`, `testify/require`, `sirupsen/logrus/hooks/test` for log assertions, `libs/atlas-constants` for shared domain types.

## Global Constraints

- **Immutable models.** Private fields + getters + `Builder`. No exported struct fields on domain models. New model fields are set through the existing builder.
- **Processors.** `Interface` + `Impl`, `NewProcessor(l, ctx)`, `var _ Processor = (*ProcessorImpl)(nil)`.
- **Multi-tenancy.** `tenant.MustFromContext(ctx)`; every registry is tenant-keyed; every Kafka command carries the tenant via producer headers.
- **Goroutines.** Never a bare `go` statement — use `routine.Go` (`tools/goroutine-guard.sh`).
- **Buff durations are MILLISECONDS** in `COMMAND_TOPIC_CHARACTER_BUFF` bodies. Never scale seconds→ms (`tools/buff-duration-guard.sh`). This task adds no buff emitters.
- **`COMMAND_TOPIC_MONSTER` `APPLY_STATUS` key set is frozen.** Add no key, rename none, retype none.
- **`nType` stays derived inside atlas-maps** and must never be added to `COMMAND_TOPIC_MIST`.
- **`SourceSkillId` on the CREATE command is the WIRE skill id**, never the `skill2.Identity`.
- **Registration is keyed on `skill2.Identity`**, never a raw wire id.
- **Reject, never clamp**, an implausible mist lifetime.
- **The mist re-apply period P (3000 ms) must stay strictly greater than** `monsterDotTickIntervalMs` (T, 1000 ms). Window = P − T = 2000 ms.
- **No value from the wire** may become a mitigation amount, magnitude, or duration.
- **No `// TODO`, no stubs, no 501s** in landed commits.
- **Repo-relative paths only** in committed files — never a literal `/home/<user>/…`.
- Generated files (`*_gen.go`, `wzsnapshot/*.json`) are **regenerated, never hand-edited**.

## File Structure

**Created**

| Path | Responsibility |
|---|---|
| `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main.go` | Turn a raw `{region,major,minor,skills,jobs}` JSON on stdin into a canonical, hash-pinned snapshot file. |
| `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main_test.go` | Golden test for canonicalisation + hash. |
| `tools/wzsnapshot-drain.sh` | Reproducible jobs-union drain of one tenant → raw snapshot JSON on stdout. |
| `tools/mist-contract-mirror-guard.sh` | Diff the two copies of the mist Kafka contract. |
| `services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast.go` | Shared mist cast: validation, caster load, CREATE emit. |
| `services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast_test.go` | Rejection table + happy path. |
| `services/atlas-channel/atlas.com/channel/skill/handler/flamegear/flamegear.go` | Flame Gear attack-cast handler. |
| `services/atlas-channel/atlas.com/channel/skill/handler/flamegear/flamegear_test.go` | Params assertions. |
| `services/atlas-channel/atlas.com/channel/skill/handler/poisonbomb/poisonbomb.go` | Poison Bomb attack-cast handler. |
| `services/atlas-channel/atlas.com/channel/skill/handler/poisonbomb/poisonbomb_test.go` | Params assertions. |
| `services/atlas-channel/atlas.com/channel/skill/handler/smokescreen/smokescreen.go` | Smokescreen USE_SKILL handler. |
| `services/atlas-channel/atlas.com/channel/skill/handler/smokescreen/smokescreen_test.go` | Params assertions. |
| `services/atlas-channel/atlas.com/channel/skill/handler/recoveryaura/recoveryaura.go` | Recovery Aura USE_SKILL handler + party snapshot. |
| `services/atlas-channel/atlas.com/channel/skill/handler/recoveryaura/recoveryaura_test.go` | Params + party snapshot assertions. |
| `services/atlas-channel/atlas.com/channel/mist/protection.go` | `Protection` model + tenant-keyed `ProtectionRegistry`. |
| `services/atlas-channel/atlas.com/channel/mist/protection_test.go` | Registry containment/expiry/removal tests. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_smoke.go` | `inProtectiveMist` predicate + its production wiring. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_smoke_test.go` | Predicate table + short-circuit test. |

**Modified**

| Path | Change |
|---|---|
| `libs/atlas-constants/gen/wzsnapshot/*.json` | Re-drained (regenerated). |
| `libs/atlas-constants/gen/wzsnapshot/PROVENANCE.md` | New drain timestamp, method note, tool references. |
| `libs/atlas-constants/skill/version_*_gen.go`, `job/version_*_gen.go`, `constants/registry_gen.go`, `skill|job/baseline_gen.go` | Regenerated. |
| `services/atlas-maps/.../kafka/message/mist/kafka.go` | 2 effect kinds, `RecoveryMp`, `PartyMemberIds`, `CreatedBody.EffectKind`. |
| `services/atlas-maps/.../mist/model.go` | `AffectedAreaTypeSmoke`, effect-kind-aware `AffectedAreaTypeFor`, `recoveryMp`/`partyMemberIds` fields + getters + `SetRecovery`. |
| `services/atlas-maps/.../mist/processor.go` | Kind validation (FR-2.5), new setters, new `AffectedAreaTypeFor` call. |
| `services/atlas-maps/.../mist/producer.go` | Carry `EffectKind` on `MIST_CREATED`. |
| `services/atlas-maps/.../character/rest.go`, `character/processor.go` | Carry `hp`; `Position` → `Snapshot`. |
| `services/atlas-maps/.../tasks/mist_tick.go` | `CharacterLookup` seam, effect-kind dispatch, `tickRecovery`, `CHANGE_MP` mirror. |
| `services/atlas-maps/atlas.com/maps/main.go` | Wire the renamed lookup. |
| `services/atlas-channel/.../kafka/message/mist/kafka.go` | Brought to a full mirror. |
| `services/atlas-channel/.../kafka/consumer/mist/consumer.go` | Populate/evict the protection registry. |
| `services/atlas-channel/.../skill/handler/poisonmist/poisonmist.go` | Refactored onto `mistcast`. |
| `services/atlas-channel/.../skill/handler/registrations/registrations.go` | Four blank imports. |
| `services/atlas-channel/.../socket/handler/character_damage.go` | `inProtectiveMist` dep + short-circuit. |
| `CLAUDE.md` | Numbered entry for the new guard. |

---

### Task 1: Snapshot drain tooling

**Files:**
- Create: `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main.go`
- Create: `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main_test.go`
- Create: `tools/wzsnapshot-drain.sh`

**Interfaces:**
- Consumes: `wzsnapshot.HashIds(skillIds []uint32, jobIds []uint16) string` (`libs/atlas-constants/gen/wzsnapshot/snapshots.go:77`).
- Produces: `mksnapshot` — reads raw JSON `{"region":string,"major":uint16,"minor":uint16,"skills":[uint32],"jobs":[uint16]}` on stdin, writes the canonical snapshot (sorted, de-duplicated, 2-space indented, `hash` filled, trailing newline) on stdout. `tools/wzsnapshot-drain.sh <tenant-id> <REGION> <major> <minor> [pod]` writes that raw JSON on stdout.

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main_test.go`:

```go
package main

import (
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/gen/wzsnapshot"

	"github.com/stretchr/testify/require"
)

// A snapshot written by mksnapshot must load cleanly through
// wzsnapshot.LoadSnapshot, which recomputes and verifies the hash. Sorting
// and de-duplication happen here so the persisted arrays match what
// LoadSnapshot hashes.
func TestCanonicalize_SortsDedupsAndPinsHash(t *testing.T) {
	raw := `{"region":"gms","major":95,"minor":1,"skills":[1002,1000,1002,8],"jobs":[200,100,100]}`

	out, err := canonicalize(strings.NewReader(raw))
	require.NoError(t, err)

	require.Equal(t, "gms", out.Region)
	require.Equal(t, uint16(95), out.Major)
	require.Equal(t, uint16(1), out.Minor)
	require.Equal(t, []uint32{8, 1000, 1002}, out.Skills)
	require.Equal(t, []uint16{100, 200}, out.Jobs)
	require.Equal(t, wzsnapshot.HashIds(out.Skills, out.Jobs), out.Hash)
}

func TestCanonicalize_RejectsEmptySkillSet(t *testing.T) {
	raw := `{"region":"gms","major":95,"minor":1,"skills":[],"jobs":[100]}`

	_, err := canonicalize(strings.NewReader(raw))
	require.Error(t, err)
	require.Contains(t, err.Error(), "empty skill set")
}

func TestRender_IsStableAndNewlineTerminated(t *testing.T) {
	raw := `{"region":"jms","major":185,"minor":1,"skills":[5],"jobs":[1]}`

	out, err := canonicalize(strings.NewReader(raw))
	require.NoError(t, err)

	b, err := render(out)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(string(b), "\n"))
	require.Contains(t, string(b), `"region": "jms"`)

	again, err := render(out)
	require.NoError(t, err)
	require.Equal(t, string(b), string(again))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants/gen && go test ./wzsnapshot/cmd/mksnapshot/ -v`
Expected: FAIL — `undefined: canonicalize`, `undefined: render`.

- [ ] **Step 3: Write the implementation**

Create `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main.go`:

```go
// Command mksnapshot turns a raw drained id-set into a canonical, hash-pinned
// wzsnapshot file.
//
// It exists so an FR-0-style re-drain is reproducible rather than a hand
// edit: wzsnapshot.LoadSnapshot recomputes the sha256 of the persisted
// arrays on every load and fails loudly on drift, so the `hash` field must
// be computed from the exact arrays that get written. Doing that by hand is
// how a snapshot silently rots.
//
// Usage:
//
//	tools/wzsnapshot-drain.sh <tenant> GMS 95 1 \
//	  | go run ./wzsnapshot/cmd/mksnapshot > wzsnapshot/gms_95_1.json
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/Chronicle20/atlas/libs/atlas-constants/gen/wzsnapshot"
)

// snapshot is the on-disk shape, field-for-field and order-for-order
// identical to wzsnapshot's private snapshotFile.
type snapshot struct {
	Region string   `json:"region"`
	Major  uint16   `json:"major"`
	Minor  uint16   `json:"minor"`
	Hash   string   `json:"hash"`
	Skills []uint32 `json:"skills"`
	Jobs   []uint16 `json:"jobs"`
}

// rawInput is the drain output: the same shape minus the hash.
type rawInput struct {
	Region string   `json:"region"`
	Major  uint16   `json:"major"`
	Minor  uint16   `json:"minor"`
	Skills []uint32 `json:"skills"`
	Jobs   []uint16 `json:"jobs"`
}

func canonicalize(r io.Reader) (snapshot, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return snapshot{}, fmt.Errorf("reading input: %w", err)
	}
	var in rawInput
	if err := json.Unmarshal(b, &in); err != nil {
		return snapshot{}, fmt.Errorf("parsing input: %w", err)
	}
	if in.Region == "" || in.Major == 0 {
		return snapshot{}, fmt.Errorf("input is missing region/major")
	}
	skills := sortedUnique32(in.Skills)
	jobs := sortedUnique16(in.Jobs)
	if len(skills) == 0 {
		return snapshot{}, fmt.Errorf("refusing to write %s %d.%d: empty skill set (drain failed?)", in.Region, in.Major, in.Minor)
	}
	if len(jobs) == 0 {
		return snapshot{}, fmt.Errorf("refusing to write %s %d.%d: empty job set (drain failed?)", in.Region, in.Major, in.Minor)
	}
	return snapshot{
		Region: in.Region,
		Major:  in.Major,
		Minor:  in.Minor,
		Hash:   wzsnapshot.HashIds(skills, jobs),
		Skills: skills,
		Jobs:   jobs,
	}, nil
}

func sortedUnique32(in []uint32) []uint32 {
	seen := make(map[uint32]bool, len(in))
	out := make([]uint32, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedUnique16(in []uint16) []uint16 {
	seen := make(map[uint16]bool, len(in))
	out := make([]uint16, 0, len(in))
	for _, v := range in {
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// render emits the snapshot with the same 2-space indentation the existing
// checked-in snapshots use, so a re-drain diffs as data rather than as
// reformatting.
func render(s snapshot) ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func main() {
	s, err := canonicalize(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mksnapshot:", err)
		os.Exit(1)
	}
	b, err := render(s)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mksnapshot:", err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(b); err != nil {
		fmt.Fprintln(os.Stderr, "mksnapshot:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants/gen && go test ./wzsnapshot/cmd/mksnapshot/ -v`
Expected: PASS (3 tests).

If `github.com/stretchr/testify` is not yet a dependency of the `gen` module, add it: `cd libs/atlas-constants/gen && go get github.com/stretchr/testify@latest && go mod tidy`.

- [ ] **Step 5: Write the drain script**

Create `tools/wzsnapshot-drain.sh` (mode 0755):

```bash
#!/usr/bin/env bash
# wzsnapshot-drain.sh — drain one tenant's skill/job id-sets from live
# atlas-data and print the raw snapshot JSON on stdout.
#
# The skills LIST endpoint (GET /api/data/skills) returns HTTP 400 in this
# baseline, so the id-set is derived by the jobs-union method documented in
# libs/atlas-constants/gen/wzsnapshot/PROVENANCE.md: drain
# GET /api/data/jobs?page[size]=200 and take the union of every row's
# attributes.skills array, with the row ids as the job set.
#
# Pipe into mksnapshot to get a canonical, hash-pinned snapshot file:
#
#   tools/wzsnapshot-drain.sh <tenant-uuid> GMS 95 1 \
#     | (cd libs/atlas-constants/gen && go run ./wzsnapshot/cmd/mksnapshot) \
#     > libs/atlas-constants/gen/wzsnapshot/gms_95_1.json
#
# Requires: kubectl context with access to namespace atlas-main, and jq.
set -euo pipefail

if [ "$#" -lt 4 ]; then
    echo "usage: $0 <tenant-uuid> <REGION> <major> <minor> [pod]" >&2
    exit 2
fi

TENANT="$1"
REGION="$2"
MAJOR="$3"
MINOR="$4"
NAMESPACE="${NAMESPACE:-atlas-main}"
POD="${5:-}"

if [ -z "$POD" ]; then
    POD="$(kubectl -n "$NAMESPACE" get pods -l app=atlas-data \
        -o jsonpath='{.items[0].metadata.name}')"
fi
if [ -z "$POD" ]; then
    echo "wzsnapshot-drain: no atlas-data pod found in namespace $NAMESPACE" >&2
    exit 1
fi

raw="$(kubectl -n "$NAMESPACE" exec "$POD" -- wget -q -O- \
    --header "TENANT_ID: $TENANT" \
    --header "REGION: $REGION" \
    --header "MAJOR_VERSION: $MAJOR" \
    --header "MINOR_VERSION: $MINOR" \
    'http://localhost:8080/api/data/jobs?page[size]=200')"

# Fail loudly rather than emitting an empty id-set: mksnapshot also rejects
# empties, but the pod name / tenant id is only known here.
pages="$(printf '%s' "$raw" | jq -r '.meta.page.last // 1')"
if [ "$pages" != "1" ]; then
    echo "wzsnapshot-drain: tenant $TENANT returned $pages pages; page[size]=200 no longer covers one page. Add a pagination loop before trusting this drain." >&2
    exit 1
fi

printf '%s' "$raw" | jq \
    --arg region "$(printf '%s' "$REGION" | tr '[:upper:]' '[:lower:]')" \
    --argjson major "$MAJOR" \
    --argjson minor "$MINOR" \
    '{region: $region, major: $major, minor: $minor,
      skills: ([.data[].attributes.skills // [] | .[]] | unique),
      jobs:   ([.data[].id | tonumber] | unique)}'
```

- [ ] **Step 6: Verify the script is syntactically sound and self-documenting**

Run: `bash -n tools/wzsnapshot-drain.sh && tools/wzsnapshot-drain.sh 2>&1 | head -2`
Expected: no syntax error; the usage line `usage: tools/wzsnapshot-drain.sh <tenant-uuid> <REGION> <major> <minor> [pod]`.

- [ ] **Step 7: Commit**

```bash
chmod +x tools/wzsnapshot-drain.sh
git add libs/atlas-constants/gen/wzsnapshot/cmd tools/wzsnapshot-drain.sh libs/atlas-constants/gen/go.mod libs/atlas-constants/gen/go.sum
git commit -m "feat(task-218): reproducible wzsnapshot drain tooling"
```

---

### Task 2: FR-0 — re-drain the snapshots and regenerate the binding tables

**Files:**
- Modify: `libs/atlas-constants/gen/wzsnapshot/gms_48_1.json`, `gms_61_1.json`, `gms_72_1.json`, `gms_79_1.json`, `gms_83_1.json`, `gms_84_1.json`, `gms_87_1.json`, `gms_92_1.json`, `gms_95_1.json`, `jms_185_1.json`, `gms_12_1.json`
- Modify: `libs/atlas-constants/gen/wzsnapshot/PROVENANCE.md`
- Modify (regenerated): `libs/atlas-constants/skill/version_*_gen.go`, `libs/atlas-constants/job/version_*_gen.go`, `libs/atlas-constants/constants/registry_gen.go`, `libs/atlas-constants/skill/baseline_gen.go`, `libs/atlas-constants/job/baseline_gen.go`

**Interfaces:**
- Consumes: Task 1's `tools/wzsnapshot-drain.sh` and `mksnapshot`.
- Produces: `skill.Identity` binding for wire id `22161003` → `EvanStage8RecoveryAura` on every version whose live atlas-data serves it. Task 12 (`recoveryaura`) depends on this; nothing else does.

**This task requires live cluster access.** If the cluster is unavailable, stop and report BLOCKED — do not fabricate a snapshot. Tasks 3–16 are independent of it.

- [ ] **Step 1: Re-list the live tenants**

Tenant ids change on reprovision, so do not reuse `design.md` §0 verbatim.

Run:
```bash
kubectl -n atlas-main exec "$(kubectl -n atlas-main get pods -l app=atlas-tenants -o jsonpath='{.items[0].metadata.name}')" \
  -- wget -q -O- 'http://localhost:8080/api/tenants' | jq -r '.data[] | [.id, .attributes.region, .attributes.majorVersion, .attributes.minorVersion] | @tsv'
```
Expected: 10 rows (GMS 48/61/72/79/83/84/87/92/95 and JMS 185). Record the mapping — it is the input to Step 2 and to the PROVENANCE table.

- [ ] **Step 2: Re-drain all 10 live tenants**

For each `(tenant, region, major, minor)` row from Step 1:

```bash
tools/wzsnapshot-drain.sh <tenant> <REGION> <major> <minor> \
  | (cd libs/atlas-constants/gen && go run ./wzsnapshot/cmd/mksnapshot) \
  > libs/atlas-constants/gen/wzsnapshot/<region>_<major>_<minor>.json
```

Then mirror gms_12 from gms_48, preserving the documented policy (identical
id-sets, only the identity header differs):

```bash
jq '.region = "gms" | .major = 12' libs/atlas-constants/gen/wzsnapshot/gms_48_1.json \
  > libs/atlas-constants/gen/wzsnapshot/gms_12_1.json
```

`hash` is a function of the id-sets only, so mirroring keeps gms_48's hash — which is exactly the existing policy.

- [ ] **Step 3: Verify the Evan range landed, and that no snapshot lost ids**

Run:
```bash
for f in libs/atlas-constants/gen/wzsnapshot/*.json; do
  printf '%s evan=%s skills=%s\n' "$(basename "$f")" \
    "$(jq '[.skills[] | select(. >= 22000000 and . < 23000000)] | length' "$f")" \
    "$(jq '.skills | length' "$f")"
done
git stash list >/dev/null; git diff --stat -- libs/atlas-constants/gen/wzsnapshot/
```
Expected: `evan` is non-zero for gms_84, gms_87, gms_92, gms_95, jms_185 and zero for gms_12/48/61/72/79/83 (this mirrors the live per-skill sweep in `design.md` §1.1). No file's `skills` count decreased — a decrease means a partial drain, not a data change; re-run that tenant before continuing.

- [ ] **Step 4: Verify the snapshots load (hash pinning)**

Run: `cd libs/atlas-constants/gen && go test ./wzsnapshot/ -run TestLoadSnapshot -v`
Expected: PASS. A hash mismatch here means `mksnapshot` was bypassed.

- [ ] **Step 5: Regenerate**

Run: `cd libs/atlas-constants/gen && go run .`
Expected: `wrote …identities_gen.go…`, `wrote 22 per-version version_*_gen.go files…`, `wrote …registry_gen.go…`.

- [ ] **Step 6: Verify `22161003` binds, on exactly the expected versions**

Run:
```bash
grep -l '22161003:' libs/atlas-constants/skill/version_*_gen.go
grep -c '^\t22[0-9]\{6\}:' libs/atlas-constants/skill/version_gms_95_1_gen.go
```
Expected: the first lists exactly `version_gms_84_1_gen.go`, `version_gms_87_1_gen.go`, `version_gms_92_1_gen.go`, `version_gms_95_1_gen.go`, `version_jms_185_1_gen.go`; the second is non-zero. If a version binds it that the Step-3 sweep says does not serve it, stop — the drain, not the expectation, is authoritative, and `prd.md` §7's availability table must be corrected in the same commit (FR-0.5).

- [ ] **Step 7: Mechanically verify the diff is additive-only (FR-0.4)**

A previously-bound wire id changing its Identity is a defect, not a result. The generated maps are one `wireId: IdentityName,` entry per line, so a removed binding shows up as a `-` line.

Run:
```bash
git diff -U0 -- libs/atlas-constants/skill/version_*_gen.go libs/atlas-constants/job/version_*_gen.go \
  | grep -E '^-' | grep -vE '^---' | grep -cE '^-[[:space:]]*[0-9]+:'
```
Expected: `0` (grep -c prints 0 and exits 1 when nothing matches — the count is what matters). Any non-zero count is a binding that disappeared or changed — investigate each line before proceeding; do not accept it because the diff is large.

- [ ] **Step 8: Drift check and guard**

Run:
```bash
cd libs/atlas-constants/gen && go run . -check && cd -
cd libs/atlas-constants && go test -race ./... && go vet ./... && cd -
tools/skill-job-id-guard.sh
```
Expected: `OK: …are up to date`, tests pass, guard exits 0.

- [ ] **Step 9: Update PROVENANCE.md**

Replace the drain-timestamp line and the Method section's opening so it records the new drain, keeping the existing gms_12 policy section and the per-version tenant table (with the Step-1 tenant ids). Add, verbatim, after the jobs-union paragraph:

```markdown
## Re-drain 2026-08 (task-218 FR-0)

Re-drained with `tools/wzsnapshot-drain.sh` piped into
`gen/wzsnapshot/cmd/mksnapshot`, which canonicalizes (sort + de-duplicate)
and recomputes the pinned `hash` — so a re-drain is reproducible rather than
a hand edit.

**Method is unchanged** (`GET /api/data/skills` still returns HTTP 400 in
this baseline; the jobs-union fallback is still required). Only the DATA is
fresh. The 2026-07-30 drain predated the Evan job documents being populated
(the `Skill.wz/Dragon/` subdirectory defect), so the union contained zero
`22xxxxxx` skills and every Evan wire id was unbound on every version.
`GET /api/data/jobs/2216/skills` now returns
`[22160000, 22161001, 22161002, 22161003]` on GMS 84/92/95 and JMS 185, and
empty on GMS 48/83 — agreeing exactly with the per-skill availability sweep
in `docs/tasks/task-218-player-cast-mists/design.md` §1.1.

The re-drain necessarily pulls in every other skill the jobs-union now
surfaces that it did not in July; that is inherent to regenerating the
snapshot wholesale. The acceptance gate is therefore that the regenerated
binding diff is ADDITIVE ONLY (no previously-bound wire id changed its
identity), not that the diff is small — task-187's divergence semantics and
the ban list behind `tools/skill-job-id-guard.sh` both depend on those
bindings.
```

Update the drain-timestamp header at the top of the file to the actual date/time of Step 2 and the pod/context used.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-constants/gen/wzsnapshot libs/atlas-constants/skill libs/atlas-constants/job libs/atlas-constants/constants
git commit -m "fix(task-218): re-drain wzsnapshots so Evan skill ids bind (FR-0)"
```

---

### Task 3: atlas-maps mist contract — two effect kinds and the recovery magnitude

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go:30-36,46-70,90-108`
- Modify: `services/atlas-maps/atlas.com/maps/mist/model.go` (fields, getters, builder)
- Modify: `services/atlas-maps/atlas.com/maps/mist/processor.go:63-114`
- Modify: `services/atlas-maps/atlas.com/maps/mist/producer.go`
- Test: `services/atlas-maps/atlas.com/maps/mist/model_test.go`, `services/atlas-maps/atlas.com/maps/mist/processor_test.go`

**Interfaces:**
- Produces (consumed by Tasks 4, 6, 7, 8, 12, 14):
  - `mistKafka.EffectKindProtection = "PROTECTION"`, `mistKafka.EffectKindRecovery = "RECOVERY"`
  - `mistKafka.CreateCommandBody.RecoveryMp int32` (`json:"recoveryMp"`), `.PartyMemberIds []uint32` (`json:"partyMemberIds"`)
  - `mistKafka.CreatedBody.EffectKind string` (`json:"effectKind"`)
  - `mist.Mist.RecoveryMp() int32`, `mist.Mist.PartyMemberIds() []uint32` (defensive copy)
  - `(*mist.Builder).SetRecovery(mp int32, partyMemberIds []uint32) *Builder`
  - `mist.ErrUnknownKind error` — returned by `Processor.Create` for an unrecognised target/effect kind

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-maps/atlas.com/maps/mist/model_test.go`:

```go
// PartyMemberIds must hand back a copy: the tick fans out across goroutines
// (processTenant), and a shared backing array is exactly the mutable state
// that parallelism punishes.
func TestMist_PartyMemberIds_IsDefensiveCopy(t *testing.T) {
	ids := []uint32{1, 2, 3}
	m := mist.NewBuilder(uuid.New(), field.NewBuilder(0, 0, 100000000).Build()).
		SetRecovery(38, ids).
		Build()

	got := m.PartyMemberIds()
	require.Equal(t, []uint32{1, 2, 3}, got)

	got[0] = 99
	ids[1] = 88
	require.Equal(t, []uint32{1, 2, 3}, m.PartyMemberIds())
}

func TestMist_RecoveryMp_RoundTripsThroughBuilder(t *testing.T) {
	m := mist.NewBuilder(uuid.New(), field.NewBuilder(0, 0, 100000000).Build()).
		SetRecovery(80, nil).
		Build()

	require.Equal(t, int32(80), m.RecoveryMp())
	require.Empty(t, m.PartyMemberIds())
}
```

Append to `services/atlas-maps/atlas.com/maps/mist/processor_test.go` (follow the file's existing helper for building a processor with a recording producer):

```go
// FR-2.5: an unrecognised kind must be REJECTED, not silently normalised to
// DISEASE — a mist that applies the wrong effect to the wrong targets is
// worse than no mist.
func TestCreate_UnknownEffectKind_RejectedAndNoMist(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	p := mist.NewProcessorWithRegistry(testLogger(t), testContext(t), rec.Provider, reg)

	_, err := p.Create(mistKafka.CreateCommandBody{
		MapId:      100000000,
		OwnerType:  mist.OwnerTypeCharacter,
		OwnerId:    1001,
		TargetKind: mistKafka.TargetKindCharacter,
		EffectKind: "TELEPORT_EVERYONE",
		Duration:   30000,
	})

	require.ErrorIs(t, err, mist.ErrUnknownKind)
	require.Empty(t, reg.AllByTenant(testTenant(t)))
	require.Empty(t, rec.Messages())
}

func TestCreate_UnknownTargetKind_RejectedAndNoMist(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	p := mist.NewProcessorWithRegistry(testLogger(t), testContext(t), rec.Provider, reg)

	_, err := p.Create(mistKafka.CreateCommandBody{
		MapId:      100000000,
		OwnerType:  mist.OwnerTypeCharacter,
		TargetKind: "NPC",
		EffectKind: mistKafka.EffectKindDisease,
		Duration:   30000,
	})

	require.ErrorIs(t, err, mist.ErrUnknownKind)
	require.Empty(t, reg.AllByTenant(testTenant(t)))
}

// FR-2.3: the pre-task-200 atlas-monsters AREA_POISON producer sends neither
// kind. That must keep working, unchanged.
func TestCreate_EmptyKinds_NormalizeToCharacterDisease(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	p := mist.NewProcessorWithRegistry(testLogger(t), testContext(t), rec.Provider, reg)

	m, err := p.Create(mistKafka.CreateCommandBody{
		MapId:     100000000,
		OwnerType: mist.OwnerTypeMonster,
		Duration:  30000,
	})

	require.NoError(t, err)
	require.Equal(t, mistKafka.TargetKindCharacter, m.TargetKind())
	require.Equal(t, mistKafka.EffectKindDisease, m.EffectKind())
}

// The recovery magnitude and the party snapshot must survive the command ->
// model hop; a dropped setter heals nobody and fails silently.
func TestCreate_CarriesRecoveryFields(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	p := mist.NewProcessorWithRegistry(testLogger(t), testContext(t), rec.Provider, reg)

	m, err := p.Create(mistKafka.CreateCommandBody{
		MapId:          100000000,
		OwnerType:      mist.OwnerTypeCharacter,
		OwnerId:        1001,
		TargetKind:     mistKafka.TargetKindCharacter,
		EffectKind:     mistKafka.EffectKindRecovery,
		RecoveryMp:     38,
		PartyMemberIds: []uint32{1001, 1002},
		Duration:       30000,
		TickIntervalMs: 3000,
	})

	require.NoError(t, err)
	require.Equal(t, int32(38), m.RecoveryMp())
	require.Equal(t, []uint32{1001, 1002}, m.PartyMemberIds())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./mist/ -v`
Expected: FAIL — `SetRecovery` / `RecoveryMp` / `ErrUnknownKind` undefined.

- [ ] **Step 3: Extend the contract**

In `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go`, replace the `EffectKind` const block comment and values:

```go
	// EffectKind selects what the mist's per-tick effect does. An empty value
	// means DISEASE. DISEASE applies a named character status via
	// COMMAND_TOPIC_CHARACTER_BUFF; DAMAGE_OVER_TIME applies a damage-bearing
	// monster status via COMMAND_TOPIC_MONSTER APPLY_STATUS; RECOVERY restores
	// MP to the party members inside via COMMAND_TOPIC_CHARACTER CHANGE_MP;
	// PROTECTION shields the owner's party from damage and is evaluated in
	// atlas-channel on the damage path -- it has no atlas-maps tick at all.
	EffectKindDisease        = "DISEASE"
	EffectKindDamageOverTime = "DAMAGE_OVER_TIME"
	EffectKindProtection     = "PROTECTION"
	EffectKindRecovery       = "RECOVERY"
```

Add to `CreateCommandBody`, after `EffectKind`:

```go
	// RecoveryMp is the per-tick MP restored by a RECOVERY mist. Unlike
	// DiseaseValue -- which is target-derived and overwritten downstream by
	// atlas-monsters -- this magnitude is caster-derived and authoritative,
	// so it gets its own field rather than overloading DiseaseValue.
	RecoveryMp int32 `json:"recoveryMp"`
	// PartyMemberIds scopes a RECOVERY mist to the caster's party, snapshot
	// at cast time by the atlas-channel handler (atlas-maps has no party
	// client). Always includes the caster. Ignored by every other kind.
	PartyMemberIds []uint32 `json:"partyMemberIds"`
```

Add to `CreatedBody`, after `SkillDelay`:

```go
	// EffectKind lets atlas-channel recognise a PROTECTION mist without
	// inferring it from the client-facing `Type` (nType) value. Carrying the
	// domain concept keeps nType a pure render detail -- see
	// mist.AffectedAreaTypeFor's doc comment.
	EffectKind string `json:"effectKind"`
```

- [ ] **Step 4: Extend the model and builder**

In `mist/model.go`, add `recoveryMp int32` and `partyMemberIds []uint32` to both `Mist` and `Builder` (after `effectKind`), add to `Build()`'s literal, and add:

```go
// RecoveryMp returns the per-tick MP a RECOVERY mist restores. 0 for every
// other effect kind.
func (m Mist) RecoveryMp() int32 {
	return m.recoveryMp
}

// PartyMemberIds returns the cast-time party snapshot a RECOVERY mist heals.
// A copy is returned: the tick fans out across goroutines (tasks.processTenant),
// so callers must not share this slice's backing array.
func (m Mist) PartyMemberIds() []uint32 {
	if len(m.partyMemberIds) == 0 {
		return nil
	}
	return append([]uint32(nil), m.partyMemberIds...)
}

// InPartySnapshot reports whether the character is in this mist's cast-time
// party snapshot. Always false for a mist with no snapshot, which is the
// correct answer for every non-RECOVERY kind.
func (m Mist) InPartySnapshot(characterId uint32) bool {
	for _, id := range m.partyMemberIds {
		if id == characterId {
			return true
		}
	}
	return false
}
```

Builder setter, next to `SetDisease`:

```go
// SetRecovery sets the RECOVERY magnitude and its cast-time party snapshot.
// Grouped rather than split because the pair is meaningless apart: a
// magnitude with no scope would heal the whole map.
func (b *Builder) SetRecovery(mp int32, partyMemberIds []uint32) *Builder {
	b.recoveryMp = mp
	b.partyMemberIds = append([]uint32(nil), partyMemberIds...)
	return b
}
```

- [ ] **Step 5: Validate kinds in `Processor.Create` and set the new fields**

In `mist/processor.go`, add above `Create`:

```go
// ErrUnknownKind is returned by Create when a command names a target or
// effect kind this service does not implement. FR-2.5: rejecting is the
// correct behaviour -- silently falling back to DISEASE would apply the
// wrong effect to the wrong targets, which is worse than creating no mist.
var ErrUnknownKind = errors.New("unknown mist target or effect kind")

func knownTargetKind(k string) bool {
	return k == mistKafka.TargetKindCharacter || k == mistKafka.TargetKindMonster
}

func knownEffectKind(k string) bool {
	switch k {
	case mistKafka.EffectKindDisease, mistKafka.EffectKindDamageOverTime,
		mistKafka.EffectKindProtection, mistKafka.EffectKindRecovery:
		return true
	}
	return false
}
```

Inside `Create`, immediately after the two normalisation blocks:

```go
	if !knownTargetKind(targetKind) {
		p.l.Warnf("Mist create rejected: unknown targetKind [%s] from owner [%s:%d] on map [%d].", body.TargetKind, body.OwnerType, body.OwnerId, body.MapId)
		return Mist{}, ErrUnknownKind
	}
	if !knownEffectKind(effectKind) {
		p.l.Warnf("Mist create rejected: unknown effectKind [%s] from owner [%s:%d] on map [%d].", body.EffectKind, body.OwnerType, body.OwnerId, body.MapId)
		return Mist{}, ErrUnknownKind
	}
```

and add `.SetRecovery(body.RecoveryMp, body.PartyMemberIds)` to the builder chain (immediately after `SetDisease`). Add `"errors"` to the import block.

- [ ] **Step 6: Carry `EffectKind` on the created event**

In `mist/producer.go`, add `EffectKind: m.EffectKind(),` to `createdEventProvider`'s `CreatedBody` literal.

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test -race ./mist/ -v`
Expected: PASS, including every pre-existing test in the package (the AREA_POISON normalisation path is unchanged).

- [ ] **Step 8: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/kafka/message/mist services/atlas-maps/atlas.com/maps/mist
git commit -m "feat(task-218): mist contract gains PROTECTION and RECOVERY effect kinds"
```

---

### Task 4: `nType` derivation for Smokescreen

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/mist/model.go:136-154`
- Modify: `services/atlas-maps/atlas.com/maps/mist/processor.go` (the `SetType` call)
- Test: `services/atlas-maps/atlas.com/maps/mist/model_test.go`

**Interfaces:**
- Consumes: `mistKafka.EffectKindProtection` (Task 3).
- Produces: `mist.AffectedAreaTypeSmoke = int32(2)`; `mist.AffectedAreaTypeFor(ownerType string, effectKind string) int32`. **This is a signature change** — the only caller is `Processor.Create`.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-maps/atlas.com/maps/mist/model_test.go`:

```go
// FR-3.4: all four outcomes, in one table. Sending 0 for a character-owned
// mist bills the caster an uninitialised nDamage (the live 1,434,803-damage
// self-hit task-200 diagnosed), and sending anything but 2 for Smokescreen
// makes the client's IsSmokeAreaByPoint lookup miss it entirely.
func TestAffectedAreaTypeFor_AllOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		ownerType  string
		effectKind string
		want       int32
	}{
		{"monster-owned disease cloud stays 0", mist.OwnerTypeMonster, mistKafka.EffectKindDisease, mist.AffectedAreaTypeMobSkill},
		{"monster-owned with empty kind stays 0", mist.OwnerTypeMonster, "", mist.AffectedAreaTypeMobSkill},
		{"monster-owned protection is still a mob area", mist.OwnerTypeMonster, mistKafka.EffectKindProtection, mist.AffectedAreaTypeMobSkill},
		{"character-owned protection is smoke", mist.OwnerTypeCharacter, mistKafka.EffectKindProtection, mist.AffectedAreaTypeSmoke},
		{"character-owned dot is a user skill area", mist.OwnerTypeCharacter, mistKafka.EffectKindDamageOverTime, mist.AffectedAreaTypeUserSkill},
		{"character-owned recovery is a user skill area", mist.OwnerTypeCharacter, mistKafka.EffectKindRecovery, mist.AffectedAreaTypeUserSkill},
		{"character-owned disease is a user skill area", mist.OwnerTypeCharacter, mistKafka.EffectKindDisease, mist.AffectedAreaTypeUserSkill},
		{"character-owned empty kind is a user skill area", mist.OwnerTypeCharacter, "", mist.AffectedAreaTypeUserSkill},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, mist.AffectedAreaTypeFor(tc.ownerType, tc.effectKind))
		})
	}
}

// The smoke value is 2 and nothing else; pinned so a future refactor cannot
// renumber it away from what the client reads.
func TestAffectedAreaTypeSmoke_IsTwo(t *testing.T) {
	require.Equal(t, int32(2), mist.AffectedAreaTypeSmoke)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./mist/ -run AffectedAreaType -v`
Expected: FAIL — `AffectedAreaTypeSmoke` undefined; `AffectedAreaTypeFor` takes 1 argument.

- [ ] **Step 3: Write the implementation**

In `mist/model.go`, extend the const block (keeping the existing long doc comment above it intact, since it already documents `== 2`):

```go
const (
	AffectedAreaTypeMobSkill  = int32(0)
	AffectedAreaTypeUserSkill = int32(1)
	// AffectedAreaTypeSmoke is 2 because the client's smoke lookup keys on it:
	// CAffectedAreaPool::IsSmokeAreaByPoint (v95 @0x434f40) rejects any area
	// whose nType != 2, and v83 CAffectedAreaPool::Update (@0x43109f) gates the
	// fade-out animation on the same value. A Smokescreen mist sent as 1 is
	// invisible to the client's own protection check.
	AffectedAreaTypeSmoke = int32(2)
)
```

Replace `AffectedAreaTypeFor`:

```go
// AffectedAreaTypeFor maps a mist's owner and effect to its nType. A
// monster-owned mist IS a mob disease cloud and must stay 0 -- that is what
// makes the client apply it to players standing in it (the pre-task-200
// AREA_POISON behaviour, which must not change). A character-owned
// PROTECTION mist is Smoke Screen (2); every other character-owned mist is a
// generic user skill area (1).
//
// Derived here rather than carried on COMMAND_TOPIC_MIST on purpose: nType is
// a client wire detail, and no producer should have to know the client's
// value table to create a mist.
func AffectedAreaTypeFor(ownerType string, effectKind string) int32 {
	if ownerType != OwnerTypeCharacter {
		return AffectedAreaTypeMobSkill
	}
	if effectKind == mistKafka.EffectKindProtection {
		return AffectedAreaTypeSmoke
	}
	return AffectedAreaTypeUserSkill
}
```

Add the `mistKafka "atlas-maps/kafka/message/mist"` import to `model.go`.

In `mist/processor.go`, update the call to `SetType(AffectedAreaTypeFor(body.OwnerType, effectKind))` — note it must use the **normalised** `effectKind` local, not `body.EffectKind`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test -race ./mist/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/mist
git commit -m "feat(task-218): derive Smokescreen nType 2 from the mist effect kind"
```

---

### Task 5: atlas-maps character lookup carries HP

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/character/rest.go`
- Modify: `services/atlas-maps/atlas.com/maps/character/processor.go`
- Modify: `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:107,240,252-281,383-390`
- Modify: `services/atlas-maps/atlas.com/maps/main.go:114-127`
- Modify (call sites only): `services/atlas-maps/atlas.com/maps/tasks/mist_tick_test.go`, `mist_tick_monster_test.go`
- Test: `services/atlas-maps/atlas.com/maps/character/processor_test.go`

**Interfaces:**
- Produces (consumed by Task 6): `tasks.CharacterLookup func(ctx context.Context, characterId uint32) (x int16, y int16, hp uint16, err error)`, replacing `tasks.PositionLookup`. `character.Processor.Snapshot(characterId uint32) (int16, int16, uint16, error)` replaces `Position`.

**Why:** FR-5.3 says recovery must not affect a dead character. Verified at plan time: `atlas-character`'s `ChangeMP` clamps to `[0, maxMP]` (`services/atlas-character/atlas.com/character/character/processor.go:1420`) but performs **no** HP check, so the dead-character half of FR-5.3 has no downstream owner. Carrying `hp` on the lookup atlas-maps already performs each tick closes it for zero extra REST calls.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-maps/atlas.com/maps/character/processor_test.go` (reuse the file's existing `httptest` server + `baseURLProvider` override helper):

```go
// The mist tick needs HP as well as position: a dead character must not be
// healed by a Recovery Aura (FR-5.3), and atlas-character's ChangeMP clamps
// to max MP but does not check HP.
func TestSnapshot_ProjectsPositionAndHp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"type":"characters","id":"1001","attributes":{"x":120,"y":-40,"hp":875}}}`))
	}))
	t.Cleanup(srv.Close)

	orig := baseURLProvider
	baseURLProvider = func() string { return srv.URL + "/api/" }
	t.Cleanup(func() { baseURLProvider = orig })

	x, y, hp, err := character.NewProcessor(testLogger(t), context.Background()).Snapshot(1001)

	require.NoError(t, err)
	require.Equal(t, int16(120), x)
	require.Equal(t, int16(-40), y)
	require.Equal(t, uint16(875), hp)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./character/ -run Snapshot -v`
Expected: FAIL — `Snapshot` undefined.

- [ ] **Step 3: Implement the projection**

In `character/rest.go`, extend the doc comment and the struct:

```go
// RestModel is the minimal projection of the atlas-character JSON:API
// resource needed by atlas-maps. atlas-character exposes many more
// attributes; only the fields we consume (position, and HP for the
// liveness check on the Recovery Aura tick) are declared here.
type RestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
	Hp uint16 `json:"hp"`
}
```

In `character/processor.go`, replace `Position` with:

```go
	// Snapshot returns the (x, y) world coordinates and current HP of the
	// character with the given id. HP is carried alongside position so the
	// mist tick can skip dead characters without a second REST call.
	// Errors propagate from the underlying REST call (e.g.
	// requests.ErrNotFound when the character does not exist).
	Snapshot(characterId uint32) (int16, int16, uint16, error)
```

```go
func (p *ProcessorImpl) Snapshot(characterId uint32) (int16, int16, uint16, error) {
	rm, err := requestById(characterId)(p.l, p.ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	return rm.X, rm.Y, rm.Hp, nil
}
```

- [ ] **Step 4: Rename the tick seam**

In `tasks/mist_tick.go`, replace the `PositionLookup` type:

```go
// CharacterLookup resolves a character's current world coordinates and HP.
// Injected as a seam so MistTick can be unit-tested without standing up the
// atlas-character REST client. HP travels with position because the recovery
// tick must skip dead characters (FR-5.3) and one REST call already carries
// both.
type CharacterLookup func(ctx context.Context, characterId uint32) (x int16, y int16, hp uint16, err error)
```

Rename the struct field `posLookup CharacterLookup` → `charLookup CharacterLookup`, update `NewMistTick(l logrus.FieldLogger, interval int, charLookup CharacterLookup) *MistTick`, and update the one call in `tickCharacters`:

```go
			x, y, _, err := r.charLookup(ctx, cid)
```

(HP is deliberately ignored on the disease path: diseasing a dead character is the pre-task-200 behaviour and is out of scope.)

In `main.go`, rename the closure and its call:

```go
	// charLookup resolves a character's world coordinates and HP via the
	// atlas-character REST client. The closure is recreated per-call so
	// each lookup runs against the caller's tenant-scoped context.
	charLookup := func(ctx context.Context, characterId uint32) (int16, int16, uint16, error) {
		return characterClient.NewProcessor(l, ctx).Snapshot(characterId)
	}
```

```go
		tasks.Register(l, rt.Context())(tasks.NewMistTick(l, 1000, charLookup))
```

- [ ] **Step 5: Update the existing tick tests' lookup stubs**

In `tasks/mist_tick_test.go` and `tasks/mist_tick_monster_test.go`, every stub of the form

```go
	posLookup := func(ctx context.Context, cid uint32) (int16, int16, error) {
		return 100, 100, nil
	}
```

becomes

```go
	charLookup := func(ctx context.Context, cid uint32) (int16, int16, uint16, error) {
		return 100, 100, 1, nil
	}
```

(HP 1 = alive; the disease path ignores it.) Update `newTestMistTick`'s parameter type to `CharacterLookup` and the `mt.posLookup` field references to `mt.charLookup`. Change **only** the signatures and the added HP value — every assertion stays as-is.

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test -race ./character/ ./tasks/ -v`
Expected: PASS. `go build ./...` clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/character services/atlas-maps/atlas.com/maps/tasks services/atlas-maps/atlas.com/maps/main.go
git commit -m "refactor(task-218): mist tick character lookup carries HP"
```

---

### Task 6: `tickRecovery` and the effect-kind dispatch

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go`
- Test: `services/atlas-maps/atlas.com/maps/tasks/mist_tick_recovery_test.go` (new)

**Interfaces:**
- Consumes: Task 3's kinds + `Mist.RecoveryMp()` / `Mist.InPartySnapshot(...)`; Task 5's `CharacterLookup`.
- Produces: `COMMAND_TOPIC_CHARACTER` `CHANGE_MP` commands, keyed on character id, with body `{"channelId":…,"amount":…}` — a key-for-key mirror of `services/atlas-channel/atlas.com/channel/kafka/message/character/kafka.go:30-35,70-73`.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-maps/atlas.com/maps/tasks/mist_tick_recovery_test.go` (reuse this package's existing `recordingProducer`, `newTestMistTick`, and tenant/field helpers from `mist_tick_test.go`):

```go
package tasks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"atlas-maps/mist"

	mistKafka "atlas-maps/kafka/message/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// recoveryMist builds a live RECOVERY mist at (100,100) whose rect covers
// (0,0)..(200,200), scoped to the given party snapshot.
func recoveryMist(t *testing.T, f field.Model, party []uint32) mist.Mist {
	t.Helper()
	return mist.NewBuilder(uuid.New(), f).
		SetOwner(mist.OwnerTypeCharacter, party[0]).
		SetOrigin(100, 100).
		SetBounds(-100, -100, 100, 100).
		SetKinds(mistKafka.TargetKindCharacter, mistKafka.EffectKindRecovery).
		SetRecovery(38, party).
		SetDuration(30 * time.Second).
		SetTickInterval(3 * time.Second).
		Build()
}

// decodeChangeMp returns the (topic, characterId, amount) triples the tick
// emitted, so the test asserts the wire shape rather than an internal call.
func decodeChangeMp(t *testing.T, rec *recordingProducer) []struct {
	CharacterId uint32
	Amount      int16
} {
	t.Helper()
	var out []struct {
		CharacterId uint32
		Amount      int16
	}
	for _, m := range rec.MessagesOn(EnvCommandTopicCharacter) {
		var env struct {
			CharacterId uint32 `json:"characterId"`
			Type        string `json:"type"`
			Body        struct {
				Amount int16 `json:"amount"`
			} `json:"body"`
		}
		require.NoError(t, json.Unmarshal(m.Value, &env))
		require.Equal(t, "CHANGE_MP", env.Type)
		out = append(out, struct {
			CharacterId uint32
			Amount      int16
		}{env.CharacterId, env.Body.Amount})
	}
	return out
}

func TestTickRecovery_HealsPartyMembersInsideOnly(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	f := field.NewBuilder(0, 0, 100000000).Build()
	tnt := testTenant(t)

	// 1001 caster inside; 1002 party inside; 1003 NON-party inside;
	// 1004 party but OUTSIDE the rect.
	m := recoveryMist(t, f, []uint32{1001, 1002, 1004})
	require.NoError(t, reg.Add(tnt, m))

	mt := newTestMistTick(t, reg, rec, func(_ context.Context, cid uint32) (int16, int16, uint16, error) {
		if cid == 1004 {
			return 900, 900, 500, nil
		}
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001, 1002, 1003, 1004} }

	mt.runOnce(context.Background())

	got := decodeChangeMp(t, rec)
	require.Len(t, got, 2)
	require.ElementsMatch(t, []uint32{1001, 1002}, []uint32{got[0].CharacterId, got[1].CharacterId})
	require.Equal(t, int16(38), got[0].Amount)
	require.Equal(t, int16(38), got[1].Amount)
}

// FR-5.3: a dead character in the cloud is not healed. atlas-character's
// ChangeMP clamps to max MP but does not check HP, so the check lives here.
func TestTickRecovery_SkipsDeadCharacter(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	f := field.NewBuilder(0, 0, 100000000).Build()
	tnt := testTenant(t)
	require.NoError(t, reg.Add(tnt, recoveryMist(t, f, []uint32{1001, 1002})))

	mt := newTestMistTick(t, reg, rec, func(_ context.Context, cid uint32) (int16, int16, uint16, error) {
		if cid == 1002 {
			return 100, 100, 0, nil
		}
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001, 1002} }

	mt.runOnce(context.Background())

	got := decodeChangeMp(t, rec)
	require.Len(t, got, 1)
	require.Equal(t, uint32(1001), got[0].CharacterId)
}

// FR-2.5 defence in depth: Processor.Create already rejects unknown kinds, so
// a mist built directly (a test, or a future producer) must not silently fall
// through to the DISEASE arm and disease everyone in the cloud.
func TestTickOneMist_UnknownEffectKind_WarnsAndEmitsNothing(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	f := field.NewBuilder(0, 0, 100000000).Build()
	tnt := testTenant(t)

	m := mist.NewBuilder(uuid.New(), f).
		SetOwner(mist.OwnerTypeCharacter, 1001).
		SetOrigin(100, 100).
		SetBounds(-100, -100, 100, 100).
		SetKinds(mistKafka.TargetKindCharacter, "TELEPORT_EVERYONE").
		SetDuration(30 * time.Second).
		SetTickInterval(3 * time.Second).
		Build()
	require.NoError(t, reg.Add(tnt, m))

	mt := newTestMistTick(t, reg, rec, func(context.Context, uint32) (int16, int16, uint16, error) {
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001} }
	hook := attachLogHook(t, mt)

	mt.runOnce(context.Background())

	require.Empty(t, rec.Messages())
	require.Contains(t, lastMessage(hook), "TELEPORT_EVERYONE")
}

// A PROTECTION mist is created with tickInterval 0, so ShouldTick is false
// and it never reaches the effect-kind switch -- it only expires. Pinned so a
// future non-zero interval cannot make it disease its own party.
func TestTickOneMist_ProtectionMistDoesNotTick(t *testing.T) {
	reg := mist.NewTestRegistry()
	rec := &recordingProducer{}
	f := field.NewBuilder(0, 0, 100000000).Build()
	tnt := testTenant(t)

	m := mist.NewBuilder(uuid.New(), f).
		SetOwner(mist.OwnerTypeCharacter, 1001).
		SetOrigin(100, 100).
		SetBounds(-100, -100, 100, 100).
		SetKinds(mistKafka.TargetKindCharacter, mistKafka.EffectKindProtection).
		SetDuration(31 * time.Second).
		SetTickInterval(0).
		Build()
	require.NoError(t, reg.Add(tnt, m))

	mt := newTestMistTick(t, reg, rec, func(context.Context, uint32) (int16, int16, uint16, error) {
		return 100, 100, 500, nil
	})
	mt.charsInField = func(tenant.Model, field.Model) []uint32 { return []uint32{1001} }

	mt.runOnce(context.Background())

	require.Empty(t, rec.Messages())
}
```

If `recordingProducer` has no `MessagesOn(topic string)` accessor, add one next to the existing `Messages()` in `mist_tick_test.go`; likewise add the small `attachLogHook` / `lastMessage` helpers using `github.com/sirupsen/logrus/hooks/test` if the package does not already have them.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-maps/atlas.com/maps && go test ./tasks/ -run 'TickRecovery|TickOneMist_' -v`
Expected: FAIL — `EnvCommandTopicCharacter` undefined and no `CHANGE_MP` emitted.

- [ ] **Step 3: Add the `CHANGE_MP` mirror**

In `tasks/mist_tick.go`, next to the existing `EnvCommandTopicMonster` const:

```go
// EnvCommandTopicCharacter is the Kafka topic where CHANGE_MP commands are
// published. Mirrors atlas-channel's value (services communicate via
// topic-name only -- no shared library import).
const EnvCommandTopicCharacter = "COMMAND_TOPIC_CHARACTER"
```

and next to the existing `monsterCommand` mirror:

```go
// characterCommand is the COMMAND_TOPIC_CHARACTER envelope, mirrored from
// atlas-channel's kafka/message/character/kafka.go Command[E]. Verified
// key-for-key against that file: {worldId, characterId, type, body}.
type characterCommand[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

// changeMpBody mirrors atlas-channel's character.ChangeMPCommandBody. Amount
// is a signed delta; atlas-character clamps the result into [0, maxMP]
// (services/atlas-character/.../character/processor.go ChangeMP ->
// enforceBounds), so no clamp is duplicated here. It does NOT check HP --
// that is why tickRecovery skips dead characters itself.
type changeMpBody struct {
	ChannelId channel.Id `json:"channelId"`
	Amount    int16      `json:"amount"`
}

// changeMpCommandProvider builds one CHANGE_MP command for a character being
// healed by a RECOVERY mist. Keyed on the character id so it lands on the
// same partition as every other command for that character.
func changeMpCommandProvider(m mist.Mist, characterId uint32) model.Provider[[]kafka.Message] {
	key := kafkaProducer.CreateKey(int(characterId))
	value := &characterCommand[changeMpBody]{
		WorldId:     m.Field().WorldId(),
		CharacterId: characterId,
		Type:        "CHANGE_MP",
		Body: changeMpBody{
			ChannelId: m.Field().ChannelId(),
			Amount:    int16(m.RecoveryMp()),
		},
	}
	return kafkaProducer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 4: Dispatch on effect kind and add `tickRecovery`**

Replace `tickOneMist`'s switch:

```go
	switch m.TargetKind() {
	case mistKafka.TargetKindMonster:
		r.tickMonsters(ctx, prov, t, m)
	default:
		// Empty target kind normalizes to CHARACTER in mist.Create; the
		// default arm also covers any mist built directly by a test.
		switch m.EffectKind() {
		case mistKafka.EffectKindRecovery:
			r.tickRecovery(ctx, prov, t, m)
		case mistKafka.EffectKindProtection:
			// Deliberate no-op. A PROTECTION mist is created with
			// tickInterval 0, so ShouldTick already returned false above and
			// this arm is unreachable today. It exists so a future non-zero
			// interval cannot fall through to the DISEASE default and start
			// diseasing everyone standing in a smoke cloud. The protection
			// itself is evaluated in atlas-channel on the damage path.
		case mistKafka.EffectKindDisease, "":
			r.tickCharacters(ctx, prov, t, m)
		default:
			r.l.Warnf("MistTick: mist [%s] has unknown effectKind [%s]; no effect applied.", m.Id(), m.EffectKind())
		}
	}
```

Add, after `tickCharacters`:

```go
// tickRecovery restores MP to every character in the mist's cast-time party
// snapshot who is alive and standing inside the rectangle.
//
// Party scoping is the snapshot carried on the CREATE command rather than a
// live lookup: atlas-maps has no party client, and giving it one would add a
// service edge for a rule nothing client-side evaluates (design §6.2). The
// cost is a staleness window bounded by the mist's 30s lifetime.
//
// The magnitude is NOT clamped here: atlas-character's ChangeMP already
// clamps into [0, maxMP] against effective max MP, and re-clamping would
// need a second REST call per character per tick to fetch a value that
// service already owns. It does not check HP, so the liveness gate is here.
func (r *MistTick) tickRecovery(ctx context.Context, prov producer.Provider, t tenant.Model, m mist.Mist) {
	if m.RecoveryMp() <= 0 {
		r.l.Warnf("MistTick: recovery mist [%s] has no magnitude; nothing to restore.", m.Id())
		return
	}
	members := r.charsInField(t, m.Field())
	if len(members) == 0 {
		return
	}
	emitErr := message.Emit(prov)(func(buf *message.Buffer) error {
		healed := 0
		for _, cid := range members {
			if !m.InPartySnapshot(cid) {
				continue
			}
			x, y, hp, err := r.charLookup(ctx, cid)
			if err != nil {
				r.l.WithError(err).Debugf("MistTick: character fetch failed for [%d].", cid)
				continue
			}
			if hp == 0 {
				continue
			}
			if !m.Contains(x, y) {
				continue
			}
			if err := buf.Put(EnvCommandTopicCharacter, changeMpCommandProvider(m, cid)); err != nil {
				return err
			}
			healed++
		}
		r.l.Debugf("MistTick: recovery mist [%s] restored %d MP to %d of %d characters in field.", m.Id(), m.RecoveryMp(), healed, len(members))
		return nil
	})
	if emitErr != nil {
		r.l.WithError(emitErr).Errorf("MistTick: failed to emit change-mp for mist [%s].", m.Id())
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-maps/atlas.com/maps && go test -race ./... && go vet ./...`
Expected: PASS, vet clean. The pre-existing `mist_tick_test.go` (disease) and `mist_tick_monster_test.go` (DoT) assertions are unchanged.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/tasks
git commit -m "feat(task-218): Recovery Aura mist tick restores party MP"
```

---

### Task 7: Full mist-contract mirror + mirror guard

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go` (replaced with a full mirror)
- Create: `tools/mist-contract-mirror-guard.sh`
- Modify: `CLAUDE.md` (Build & Verification list)
- Test: `services/atlas-channel/atlas.com/channel/mist/producer_test.go` (existing — must still pass)

**Interfaces:**
- Produces (consumed by Tasks 8–12, 14): the channel-side `mistmsg` package gains `EffectKindProtection`, `EffectKindRecovery`, `CreateCommandBody.RecoveryMp`, `.PartyMemberIds`, `CreatedBody.EffectKind`, plus the previously-missing `CommandTypeCancel`, `CancelCommandBody`, `ReasonExpired`, `ReasonCancelled`.

**Why a full mirror:** the two files live in separate Go modules, so a json tag changed in one and not the other compiles clean and decodes into a zero-valued body at runtime. The channel copy is currently a *partial* mirror, which is how divergence starts. This task adds three keys to the contract — the moment the hazard is highest.

- [ ] **Step 1: Write the failing guard**

Create `tools/mist-contract-mirror-guard.sh` (mode 0755):

```bash
#!/usr/bin/env bash
# mist-contract-mirror-guard.sh — enforces that the COMMAND_TOPIC_MIST /
# EVENT_TOPIC_MIST contract is identical in its two copies.
#
# atlas-maps owns the contract; atlas-channel carries a mirror because the two
# services live in separate Go modules and nothing in the compiler links them.
# A field name or json tag changed in one copy and not the other does not fail
# any build — it decodes into a zero-valued body at runtime, silently: a mist
# with no bounds, no lifetime, or (task-218) no recovery magnitude and no party
# scope. Modelled on tools/trade-contract-mirror-guard.sh, which exists for the
# same failure mode on a different pair of modules.
#
# The two files are compared from their `package` clause onward: the only
# permitted difference is the leading doc comment, which names the mirror
# direction and therefore differs by design.
#
# Run from the repo root; drift → non-zero exit. task-218.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OWNER="$ROOT/services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go"
MIRROR="$ROOT/services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go"

rc=0
for f in "$OWNER" "$MIRROR"; do
    if [ ! -f "$f" ]; then
        echo "mist-contract-mirror-guard: FAIL — missing contract file: ${f#"$ROOT"/}"
        rc=1
    fi
done
[ "$rc" -ne 0 ] && exit "$rc"

body() { awk '/^package /{p=1} p' "$1"; }

if diff -u --label "owner: ${OWNER#"$ROOT"/}" --label "mirror: ${MIRROR#"$ROOT"/}" \
        <(body "$OWNER") <(body "$MIRROR"); then
    echo "OK: the mist contract mirror matches its owner."
    exit 0
fi

echo ""
echo "mist-contract-mirror-guard: FAIL — the mist Kafka contract has drifted."
echo "  owner : ${OWNER#"$ROOT"/}"
echo "  mirror: ${MIRROR#"$ROOT"/}"
echo ""
echo "  These two files are one cross-service wire contract. Struct names, field"
echo "  names and json tags must match exactly; only the leading doc comment,"
echo "  which names the mirror direction, may differ."
echo ""
echo "  FIX: apply the change to the owner, then re-copy it to the mirror and"
echo "  restore the mirror's doc-comment header:"
echo "    cp ${OWNER#"$ROOT"/} ${MIRROR#"$ROOT"/}"
echo "  (then edit the copied header back to \"Mirrors <owner path>\")"
exit 1
```

- [ ] **Step 2: Run the guard to verify it fails**

Run: `chmod +x tools/mist-contract-mirror-guard.sh && tools/mist-contract-mirror-guard.sh`
Expected: FAIL with a diff — the channel copy is missing `CommandTypeCancel`, `CancelCommandBody`, `ReasonExpired`/`ReasonCancelled`, and orders its declarations differently.

- [ ] **Step 3: Bring the mirror to full parity**

```bash
cp services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go \
   services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go
```

Then prepend the mirror-direction header above the `package` clause of the channel copy (this is the one sanctioned difference):

```go
// Package mist mirrors atlas-maps' COMMAND_TOPIC_MIST / EVENT_TOPIC_MIST
// contract (services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go).
//
// atlas-maps OWNS this contract; this file is a copy because the two services
// live in separate Go modules and nothing in the compiler links them. Do not
// edit this file directly: change the owner, re-copy, and restore this header.
// tools/mist-contract-mirror-guard.sh fails CI on any other difference.
```

Both files declare `package mist` and import the same three `atlas-constants` packages plus `uuid`, so the copy compiles unchanged in the channel module.

- [ ] **Step 4: Run the guard and the channel tests**

Run:
```bash
tools/mist-contract-mirror-guard.sh
cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./mist/... ./kafka/... && cd -
```
Expected: `OK: the mist contract mirror matches its owner.`; build clean; `TestCreateCommandProvider_KeySetMatchesAtlasMaps` still passes.

- [ ] **Step 5: Register the guard in CLAUDE.md**

Append to the numbered Build & Verification list in `CLAUDE.md`, after item 13:

```markdown
14. **`tools/mist-contract-mirror-guard.sh` clean from the repo root** whenever
    either copy of the mist Kafka contract changed. atlas-maps owns
    `kafka/message/mist/kafka.go`; atlas-channel carries a mirror, and the two
    live in separate Go modules, so a field name or json tag changed in one and
    not the other fails no build — it decodes into a zero-valued body at
    runtime, silently: a mist with no bounds, no lifetime, or no recovery
    magnitude and no party scope (task-218). The guard diffs the two files from
    their `package` clause onward; only the leading doc comment, which names the
    mirror direction, may differ.
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/mist tools/mist-contract-mirror-guard.sh CLAUDE.md
git commit -m "feat(task-218): full mist contract mirror plus a mirror guard"
```

---

### Task 8: `mistcast` — the shared cast helper

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast_test.go`

**Interfaces:**
- Consumes: `effect.Model` (`Duration() int32`, `LT()/RB() point.Model`, `X() int16`), `mistmsg.CreateCommandBody` (Task 7), `mist.NewProcessor(l, ctx).Create(body)`, `character.NewProcessor(l, ctx).GetById()(id)` → `X()/Y()`.
- Produces (consumed by Tasks 9–12):
  - `mistcast.PlayerMistTickIntervalMs int64 = 3000`
  - `mistcast.MaxPlayerMistDurationMs int32 = 300_000`
  - `mistcast.Params{SkillName, TargetKind, EffectKind, Disease string; TickMs int64; RecoveryMp int32; PartyMemberIds []uint32}`
  - `mistcast.Seams{LoadCaster func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error); EmitCreate func(logrus.FieldLogger, context.Context, mistmsg.CreateCommandBody) error}`
  - `mistcast.DefaultLoadCaster`, `mistcast.DefaultEmitCreate` (the production seam values)
  - `mistcast.Cast(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, skillId skill2.Id, skillLevel byte, e effect.Model, p Params, s Seams) error` — returns nil on every rejection path.

**Why `Seams` is a parameter rather than package state:** each handler keeps its own `loadCaster`/`emitCreate` package vars, which is the seam idiom the existing `poisonmist` tests already drive (`poisonmist_test.go:52-62`). Passing them in lets Task 9 refactor `poisonmist` onto this helper with its tests **unmodified**, which is the PRD's regression bar.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast_test.go`:

```go
package mistcast

import (
	"context"
	"testing"

	"atlas-channel/data/skill/effect"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// stubEffect builds an effect.Model with only the fields a mist cast reads.
// effect.Model has unexported fields and no exported constructor; hydrating
// through effect.Extract on a RestModel literal is the supported
// construction path (same helper shape as poisonmist_test.go:38-47).
func stubEffect(duration int32, ltX, ltY, rbX, rbY int16) effect.Model {
	return stubEffectX(duration, 0, ltX, ltY, rbX, rbY)
}

// stubEffectX additionally sets the `x` node, which Recovery Aura reads as
// its per-tick MP magnitude.
func stubEffectX(duration int32, x int16, ltX, ltY, rbX, rbY int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: duration,
		X:        x,
		LT:       &effect.PointRestModel{X: ltX, Y: ltY},
		RB:       &effect.PointRestModel{X: rbX, Y: rbY},
	})
	if err != nil {
		panic(err)
	}
	return m
}

func dotParams() Params {
	return Params{
		SkillName:  "Test Mist",
		TargetKind: mistmsg.TargetKindMonster,
		EffectKind: mistmsg.EffectKindDamageOverTime,
		Disease:    "POISON",
		TickMs:     PlayerMistTickIntervalMs,
	}
}

// run drives Cast with recording seams and returns everything emitted plus
// the log hook, so each case asserts on the wire body and the warning.
func run(t *testing.T, e effect.Model, p Params) ([]mistmsg.CreateCommandBody, *test.Hook) {
	t.Helper()
	l, hook := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody
	s := Seams{
		LoadCaster: func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
			return 250, -80, nil
		},
		EmitCreate: func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
			emitted = append(emitted, body)
			return nil
		},
	}
	f := field.NewBuilder(0, 0, 100000000).Build()
	require.NoError(t, Cast(l, context.Background(), f, 1001, 2111003, 20, e, p, s))
	return emitted, hook
}

func TestCast_HappyPath_EmitsExactlyOneCreate(t *testing.T) {
	emitted, _ := run(t, stubEffect(40000, -110, -82, 110, 83), dotParams())

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, "CHARACTER", b.OwnerType)
	require.Equal(t, uint32(1001), b.OwnerId)
	require.Equal(t, int16(250), b.OriginX)
	require.Equal(t, int16(-80), b.OriginY)
	require.Equal(t, int16(-110), b.LtX)
	require.Equal(t, int16(83), b.RbY)
	require.Equal(t, int64(40000), b.Duration)
	require.Equal(t, int64(40000), b.DiseaseDuration)
	require.Equal(t, PlayerMistTickIntervalMs, b.TickIntervalMs)
	require.Equal(t, mistmsg.TargetKindMonster, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindDamageOverTime, b.EffectKind)
	require.Equal(t, "POISON", b.Disease)
	// Target-derived magnitude: atlas-monsters resolves and overwrites it.
	require.Equal(t, int32(0), b.DiseaseValue)
	// The WIRE id, not the Identity: the client picks its rendering arm from it.
	require.Equal(t, uint32(2111003), b.SourceSkillId)
	require.Equal(t, uint32(20), b.SourceSkillLevel)
}

func TestCast_ZeroLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(0, -110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "no lifetime")
	require.Contains(t, hook.LastEntry().Message, "Test Mist")
}

func TestCast_LifetimeShorterThanOneTick_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(int32(PlayerMistTickIntervalMs)-1, -110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "shorter than one tick")
}

func TestCast_LifetimeEqualToOneTick_Accepted(t *testing.T) {
	emitted, _ := run(t, stubEffect(int32(PlayerMistTickIntervalMs), -110, -82, 110, 83), dotParams())
	require.Len(t, emitted, 1)
}

// A PROTECTION mist never ticks, so the sub-tick gate must not apply to it.
// Smokescreen's real lifetime (31s) clears it anyway; this pins the rule.
func TestCast_ZeroTickInterval_SkipsSubTickGate(t *testing.T) {
	p := Params{
		SkillName:  "Smoke Test",
		TargetKind: mistmsg.TargetKindCharacter,
		EffectKind: mistmsg.EffectKindProtection,
		TickMs:     0,
	}
	emitted, _ := run(t, stubEffect(1000, -110, -82, 110, 83), p)
	require.Len(t, emitted, 1)
	require.Equal(t, int64(0), emitted[0].TickIntervalMs)
	require.Equal(t, "", emitted[0].Disease)
}

func TestCast_DegenerateRectangle_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(40000, 110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "degenerate rectangle")
}

// FR-8.2: reject, never truncate. The client computes its own tEnd from its
// own WZ, so a server-side clamp desynchronises it.
func TestCast_ImplausibleLifetime_Rejected(t *testing.T) {
	emitted, hook := run(t, stubEffect(MaxPlayerMistDurationMs+1, -110, -82, 110, 83), dotParams())
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "implausible lifetime")
}

func TestCast_CasterLoadFailure_EmitsNothingAndReturnsNil(t *testing.T) {
	l, hook := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody
	s := Seams{
		LoadCaster: func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
			return 0, 0, errors.New("boom")
		},
		EmitCreate: func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
			emitted = append(emitted, body)
			return nil
		},
	}
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Cast(l, context.Background(), f, 1001, 2111003, 20, stubEffect(40000, -110, -82, 110, 83), dotParams(), s))
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "failed to load caster")
}

func TestCast_EmitFailure_ReturnsNil(t *testing.T) {
	l, hook := test.NewNullLogger()
	s := Seams{
		LoadCaster: func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
			return 0, 0, nil
		},
		EmitCreate: func(logrus.FieldLogger, context.Context, mistmsg.CreateCommandBody) error {
			return errors.New("kafka down")
		},
	}
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Cast(l, context.Background(), f, 1001, 2111003, 20, stubEffect(40000, -110, -82, 110, 83), dotParams(), s))
	require.Contains(t, hook.LastEntry().Message, "failed to emit CREATE")
}

// FR-5: a RECOVERY cast carries its magnitude and party snapshot explicitly;
// DiseaseValue is never overloaded for it.
func TestCast_RecoveryParams_CarryMagnitudeAndParty(t *testing.T) {
	p := Params{
		SkillName:      "Recovery Test",
		TargetKind:     mistmsg.TargetKindCharacter,
		EffectKind:     mistmsg.EffectKindRecovery,
		TickMs:         PlayerMistTickIntervalMs,
		RecoveryMp:     38,
		PartyMemberIds: []uint32{1001, 1002},
	}
	emitted, _ := run(t, stubEffect(30000, -200, -125, 200, 30), p)

	require.Len(t, emitted, 1)
	require.Equal(t, int32(38), emitted[0].RecoveryMp)
	require.Equal(t, []uint32{1001, 1002}, emitted[0].PartyMemberIds)
	require.Equal(t, int32(0), emitted[0].DiseaseValue)
	require.Equal(t, "", emitted[0].Disease)
}
```

Add the `"errors"` import. `effect.RestModel`'s relevant fields are `Duration int32` (:—), `X int16` (`rest.go:41`), and `LT`/`RB *PointRestModel` (`rest.go:63-64`) — there is **no** `effect.NewBuilder`, so every stub in this task and Tasks 10–12 goes through `effect.Extract`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/mistcast/ -v`
Expected: FAIL — package has no non-test Go files.

- [ ] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/skill/handler/mistcast/mistcast.go`:

```go
// Package mistcast holds the cast-time logic every player-cast mist skill
// shares: validate the effect, load the caster's position, emit CREATE to
// atlas-maps. The five mist skills (Poison Mist, Flame Gear, Poison Bomb,
// Smokescreen, Recovery Aura) differ only in the Params they build, so the
// validation rules -- each of which encodes an expensively-learned client
// behaviour -- live here once rather than in five copies.
package mistcast

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/mist"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// PlayerMistTickIntervalMs is the RE-APPLY cadence of a player-cast mist --
// how often atlas-maps re-issues its per-tick effect to everything still
// inside the cloud (mist.Mist.ShouldTick). It is NOT the DoT damage cadence
// and does NOT flow into atlas-monsters' DoT gate.
//
// atlas-maps sends a SEPARATE, strictly smaller DoT tick interval on every
// APPLY_STATUS command (monsterDotTickIntervalMs,
// services/atlas-maps/atlas.com/maps/tasks/mist_tick.go, currently 1000ms).
// atlas-monsters' StatusEffect.ShouldTick gates actual damage on
// `since(lastTick) >= tickInterval`
// (services/atlas-monsters/atlas.com/monsters/monster/status.go:129-134).
//
// It must be an interval, not a WZ value: no `dotInterval` node exists in any
// provisioned Skill.wz for these skills (task-200 design §2.1; task-218
// design §6.3 for Recovery Aura, whose dot/dotInterval/dotTime are all 0).
//
// This value MUST exceed the DoT tick interval sent by atlas-maps.
// atlas-monsters' ModelBuilder.AddStatusEffect REPLACES a same-type POISON on
// every re-apply with a fresh StatusEffect whose lastTick = now
// (services/atlas-monsters/atlas.com/monsters/monster/builder.go:141-163,
// services/atlas-monsters/atlas.com/monsters/monster/status.go:35-49). So the
// eligible damage window per re-apply cycle is `PlayerMistTickIntervalMs (P)
// - monsterDotTickIntervalMs (T)` wide, NOT P. A prior fix attempt set
// atlas-maps' emitted TickInterval to this same constant (P == T by
// construction), which makes that window exactly 0 regardless of P's value --
// the mist would never deal damage no matter how this constant was tuned.
// With P = 3000ms and T = 1000ms the window is a genuine 2000ms per cycle.
const PlayerMistTickIntervalMs int64 = 3000

// MaxPlayerMistDurationMs rejects (never truncates) an implausible mist
// lifetime. The largest legitimate `time` across all five mist skills and all
// provisioned versions is Smokescreen's 60s at level 30, so this 5-minute
// ceiling is 5x the largest real value and can only fire on corrupt or
// mis-scaled data.
//
// This is deliberately NOT atlas-monsters' 60s MistDurationCapMs. A clamp
// would desynchronise the client, which computes its own
// tEnd = tStart + 1000*SKILLLEVELDATA::tTime from its own WZ (v83 @0x43200f,
// v95 @0x437c95) and would keep rendering a mist the server stopped ticking.
const MaxPlayerMistDurationMs int32 = 300_000

// Params is everything a mist cast differs by. Everything else -- the
// rectangle, the lifetime, the origin, the source ids -- is derived from the
// effect and the cast itself.
type Params struct {
	// SkillName names the skill in log lines. Rejections must be traceable
	// to a skill without decoding an id.
	SkillName string
	// TargetKind / EffectKind are the mist contract's descriptors.
	TargetKind string
	EffectKind string
	// Disease is the status name a DAMAGE_OVER_TIME mist applies; empty for
	// every other kind.
	Disease string
	// TickMs is the mist's re-apply cadence. 0 means "never ticks", which is
	// how a PROTECTION mist is expressed -- it is evaluated on the channel's
	// damage path, not by an atlas-maps tick.
	TickMs int64
	// RecoveryMp is the per-tick MP a RECOVERY mist restores; 0 otherwise.
	RecoveryMp int32
	// PartyMemberIds scopes a RECOVERY mist; nil otherwise.
	PartyMemberIds []uint32
}

// Seams are the two external effects a cast performs. Each handler keeps its
// own package-level vars and passes them here so its tests can record
// instead of hitting the character service or Kafka.
type Seams struct {
	LoadCaster func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error)
	EmitCreate func(l logrus.FieldLogger, ctx context.Context, body mistmsg.CreateCommandBody) error
}

// DefaultLoadCaster returns the caster's (X, Y) from the character service.
var DefaultLoadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, err
	}
	return c.X(), c.Y(), nil
}

// DefaultEmitCreate publishes the CREATE command to atlas-maps.
var DefaultEmitCreate = func(l logrus.FieldLogger, ctx context.Context, body mistmsg.CreateCommandBody) error {
	return mist.NewProcessor(l, ctx).Create(body)
}

// Cast validates the effect, loads the caster, and emits CREATE.
//
// Every rejection returns nil and emits nothing: there is no MP or cooldown
// rollback path, by design (task-200 FR-3.2 / FR-6.5). By the time this runs
// the cost has already been charged -- by processAttack for the attack-cast
// skills, by UseSkill for the USE_SKILL ones.
func Cast(
	l logrus.FieldLogger,
	ctx context.Context,
	f field.Model,
	characterId uint32,
	skillId skill2.Id,
	skillLevel byte,
	e effect.Model,
	p Params,
	s Seams,
) error {
	duration := e.Duration()
	lt, rb := e.LT(), e.RB()

	if duration <= 0 {
		l.Warnf("%s: rejected cast by [%d] — no lifetime (effect duration %d ms).", p.SkillName, characterId, duration)
		return nil
	}
	// A mist that never ticks (PROTECTION) has no sub-tick floor to clear.
	if p.TickMs > 0 && int64(duration) < p.TickMs {
		l.Warnf("%s: rejected cast by [%d] — lifetime shorter than one tick (%d ms < %d ms).", p.SkillName, characterId, duration, p.TickMs)
		return nil
	}
	if rb.X() <= lt.X() || rb.Y() <= lt.Y() {
		l.Warnf("%s: rejected cast by [%d] — degenerate rectangle lt(%d,%d) rb(%d,%d).", p.SkillName, characterId, lt.X(), lt.Y(), rb.X(), rb.Y())
		return nil
	}
	if duration > MaxPlayerMistDurationMs {
		l.Warnf("%s: rejected cast by [%d] — implausible lifetime (%d ms > %d ms ceiling).", p.SkillName, characterId, duration, MaxPlayerMistDurationMs)
		return nil
	}

	x, y, err := s.LoadCaster(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Errorf("%s: failed to load caster [%d]; no mist created.", p.SkillName, characterId)
		return nil
	}

	body := mistmsg.CreateCommandBody{
		WorldId:    f.WorldId(),
		ChannelId:  f.ChannelId(),
		MapId:      f.MapId(),
		Instance:   f.Instance(),
		OwnerType:  "CHARACTER",
		OwnerId:    characterId,
		TargetKind: p.TargetKind,
		EffectKind: p.EffectKind,
		OriginX:    x,
		OriginY:    y,
		LtX:        int16(lt.X()),
		LtY:        int16(lt.Y()),
		RbX:        int16(rb.X()),
		RbY:        int16(rb.Y()),
		Disease:    p.Disease,
		// Magnitude 0 is correct, not a shortcut: the POISON magnitude is
		// TARGET-derived, so only atlas-monsters can fill it in. It resolves
		// the value per monster at apply time as
		// ceil(maxHP/(70 - sourceSkillLevel)) capped at 32767
		// (monster.ResolvePoisonDamage), and that single value is both the
		// damage each tick applies and the magnitude the client renders its
		// own tick numbers from. Anything sent here would be overwritten.
		DiseaseValue: 0,
		// Per-target duration = the mist's lifetime. With no WZ `dotTime`,
		// this is the value that matches observable behaviour, and
		// atlas-monsters REPLACES a same-type status on re-apply, so a
		// monster inside the cloud simply has its expiry pushed forward.
		DiseaseDuration: int64(duration),
		Duration:        int64(duration),
		TickIntervalMs:  p.TickMs,
		RecoveryMp:      p.RecoveryMp,
		PartyMemberIds:  p.PartyMemberIds,
		// The WIRE skill id, deliberately -- not the resolved Identity. The
		// client compares this against its own WZ to pick the rendering arm
		// (CAffectedAreaPool::AffectedAreaAnimationCreated, v83 @0x431d50,
		// v95 @0x437515), so it must be the id that version binds. This is
		// the one place a raw wire id is the correct value.
		SourceSkillId:    uint32(skillId),
		SourceSkillLevel: uint32(skillLevel),
	}

	if err := s.EmitCreate(l, ctx, body); err != nil {
		l.WithError(err).Errorf("%s: failed to emit CREATE for character [%d].", p.SkillName, characterId)
		return nil
	}

	l.Infof("%s: character [%d] cast level [%d] at (%d,%d), rect lt(%d,%d) rb(%d,%d), lifetime %d ms.",
		p.SkillName, characterId, skillLevel, x, y, lt.X(), lt.Y(), rb.X(), rb.Y(), duration)
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/mistcast/ -v`
Expected: PASS (11 tests).

- [ ] **Step 5: Assert the FR-6.4 window in a test that owns it**

Append to `mistcast_test.go`:

```go
// FR-6.4: the re-apply period P must strictly exceed the DoT tick interval T
// that atlas-maps emits, or the eligible damage window is exactly zero and
// the mist deals no damage at any tuning. T is atlas-maps'
// monsterDotTickIntervalMs; it is duplicated here as a literal on purpose --
// the two services share no module, and this test is what catches a change
// to either side.
func TestPlayerMistTickInterval_LeavesANonZeroDamageWindow(t *testing.T) {
	const monsterDotTickIntervalMs int64 = 1000 // atlas-maps tasks/mist_tick.go

	require.Greater(t, PlayerMistTickIntervalMs, monsterDotTickIntervalMs)
	require.Equal(t, int64(2000), PlayerMistTickIntervalMs-monsterDotTickIntervalMs)
}
```

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/mistcast/ -run Window -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/mistcast
git commit -m "feat(task-218): shared mistcast validation and emit helper"
```

---

### Task 9: Refactor `poisonmist` onto `mistcast`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist.go`
- Test: `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist_test.go` — **must not be edited**

**Interfaces:**
- Consumes: `mistcast.Cast`, `mistcast.Params`, `mistcast.Seams`, `mistcast.DefaultLoadCaster`, `mistcast.DefaultEmitCreate`.
- Produces: no new exported surface. `poisonmist.PlayerMistTickIntervalMs` and `poisonmist.MaxPlayerMistDurationMs` remain, as aliases of the `mistcast` values, because the existing tests reference them.

**The regression bar:** `poisonmist_test.go` passes **unmodified**. It swaps the package-level `loadCaster` / `emitCreate` vars and reads `PlayerMistTickIntervalMs` / `MaxPlayerMistDurationMs`, so all four identifiers must survive the refactor with the same meaning.

- [ ] **Step 1: Run the existing tests and record the baseline**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/poisonmist/ -v 2>&1 | tail -20`
Expected: PASS. Note the test names — the same set must pass at Step 4.

- [ ] **Step 2: Rewrite the handler over `mistcast`**

Replace the body of `services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist.go` below the package doc comment with:

```go
package poisonmist

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/skill/handler/mistcast"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Poison Mist is registered on the ATTACK-cast registry, not the use-skill
// one. The client delivers 2111003 on a magic-attack packet (opcode 0x2E at
// GMS v83) because the skill carries `damage`/`attackCount`/`mobCount` in
// Skill.wz -- verified live: `GET /api/data/skills/2111003` returns
// damage 100, attackCount 1, mobCount 1, prop 0.41. It never arrives on
// USE_SKILL, so a channelhandler.Register here would never fire (and would
// additionally suppress the generic MP cost -- see AttackCastHandler's doc).
func init() {
	channelhandler.RegisterAttackCast(skill2.FirePoisonMagicianPoisonMist, Apply)
}

// The cast-time constants are owned by mistcast, where their full rationale
// lives (the P > T damage-window invariant, and reject-don't-clamp). Kept
// visible here because this package's tests -- the regression bar for the
// whole mist family -- assert against them by these names.
const (
	PlayerMistTickIntervalMs = mistcast.PlayerMistTickIntervalMs
	MaxPlayerMistDurationMs  = mistcast.MaxPlayerMistDurationMs
)

// loadCaster / emitCreate are this handler's copies of the mistcast seams.
// Package-level vars so tests can record instead of calling the character
// service and Kafka.
var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// Apply is the Poison Mist handler installed in the per-skill attack-cast
// registry.
//
// By the time it runs, processAttack has already charged MP, applied the
// direct magic-attack damage, and broadcast the attack. Every rejection
// inside mistcast.Cast returns nil and emits nothing: there is no MP or
// cooldown rollback path, by design (FR-3.2 / FR-6.5).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	skillId skill2.Id,
	skillLevel byte,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		skillId skill2.Id,
		skillLevel byte,
		e effect.Model,
	) error {
		return func(
			_ writer.Producer,
			f field.Model,
			characterId uint32,
			skillId skill2.Id,
			skillLevel byte,
			e effect.Model,
		) error {
			return mistcast.Cast(l, ctx, f, characterId, skillId, skillLevel, e,
				mistcast.Params{
					SkillName: "Poison Mist",
					// A player-cast mist targets MONSTERS with a
					// damage-bearing status, unlike the monster AREA_POISON
					// mist which diseases CHARACTERS.
					TargetKind: mistmsg.TargetKindMonster,
					EffectKind: mistmsg.EffectKindDamageOverTime,
					Disease:    "POISON",
					TickMs:     mistcast.PlayerMistTickIntervalMs,
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
```

- [ ] **Step 3: Verify the test file is untouched**

Run: `git diff --stat -- services/atlas-channel/atlas.com/channel/skill/handler/poisonmist/poisonmist_test.go`
Expected: **empty output**. If the test needed editing, the refactor is wrong — fix the handler, not the test.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/... -v`
Expected: PASS — the same Poison Mist test names as Step 1, plus mistcast's.

If a log-message assertion fails, it is because `mistcast.Cast` formats the skill name via `%s` where the old code hard-coded `"Poison Mist: "`. The format strings in Task 8 Step 3 reproduce the originals exactly with `p.SkillName = "Poison Mist"` — re-check the punctuation rather than editing the test.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/poisonmist
git commit -m "refactor(task-218): poisonmist casts through mistcast"
```

---

### Task 10: Flame Gear and Poison Bomb handlers

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/flamegear/flamegear.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/flamegear/flamegear_test.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/poisonbomb/poisonbomb.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/poisonbomb/poisonbomb_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `mistcast.Cast/Params/Seams/Default*`, `channelhandler.RegisterAttackCast`, `skill2.BlazeWizardStage3FlameGear`, `skill2.NightWalkerStage3PoisonBomb`.
- Produces: `flamegear.Apply`, `poisonbomb.Apply` — both of type `channelhandler.AttackCastHandler`.

**Registry evidence (FR-7.2), both attack-delivered:** Flame Gear serves `prop` 0.51→0.80 on gms 72/79/83/84/87/92 and jms 185, plus `mobCount` 8 / `damage` 124 at v95. Poison Bomb serves `mobCount` 6 and `damage` 104→220 on every version. Both trip the discriminator (`prop != 0` or `mobCount != 1` or `damage != 100`), so `processAttack` charges their MP as it does for Poison Mist. See `design.md` §4.

**Status (FR-6.3):** Flame Gear applies `POISON`, magnitude 0. `monsterStatus` is `{}` for 12111005 on every version served; `atlas-monsters` has exactly two DoT statuses (`StatusPoison`, `StatusVenom` — `monster/status.go:20,23`) and `VENOM` is the caster-magnitude Night Lord stat. The only caster-side candidate, `dot`, is 0 on six of the seven versions that bind the skill. See `design.md` §1.4.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/flamegear/flamegear_test.go`:

```go
package flamegear

import (
	"context"
	"testing"

	"atlas-channel/data/skill/effect"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/skill/handler/mistcast"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// FR-7.1/FR-7.3: the client delivers Flame Gear on an ATTACK packet, so it
// must be on the attack-cast registry and NOT on the use-skill one --
// registering it there would never fire AND would silently zero its MP cost.
// Registration is by Identity so one call covers every version that binds it.
func TestInit_RegistersOnAttackCastRegistryByIdentity(t *testing.T) {
	_, ok := channelhandler.LookupAttackCast(skill2.BlazeWizardStage3FlameGear)
	require.True(t, ok, "Flame Gear must be registered on the attack-cast registry")

	_, wrong := channelhandler.Lookup(skill2.BlazeWizardStage3FlameGear)
	require.False(t, wrong, "Flame Gear must NOT be on the use-skill registry")
}

func TestApply_EmitsMonsterDotMistWithWireSkillId(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		return 300, 120, nil
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e := stubEffect(40000, -200, -250, 200, 30)
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(12111005), 30, e))

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, mistmsg.TargetKindMonster, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindDamageOverTime, b.EffectKind)
	require.Equal(t, "POISON", b.Disease)
	// Target-derived: atlas-monsters resolves and overwrites it (FR-6.3).
	require.Equal(t, int32(0), b.DiseaseValue)
	require.Equal(t, mistcast.PlayerMistTickIntervalMs, b.TickIntervalMs)
	// FR-7.4: the WIRE id, not the Identity.
	require.Equal(t, uint32(12111005), b.SourceSkillId)
	require.Equal(t, int16(300), b.OriginX)
	require.Equal(t, int32(0), b.RecoveryMp)
	require.Nil(t, b.PartyMemberIds)
}

// The shortest legitimate lifetime (4000ms at L1 on gms 72-92/jms) clears
// the sub-tick gate; pinned so a cadence change cannot silently disable the
// skill at low levels.
func TestApply_ShortestRealLifetimeIsAccepted(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) { return 0, 0, nil }
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e := stubEffect(4000, -200, -250, 200, 30)
	f := field.NewBuilder(0, 0, 100000000).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, skill2.Id(12111005), 1, e))
	require.Len(t, emitted, 1)
}

// stubEffect hydrates an effect.Model through the supported construction
// path -- effect.Model has unexported fields and no exported builder.
func stubEffect(duration int32, ltX, ltY, rbX, rbY int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: duration,
		LT:       &effect.PointRestModel{X: ltX, Y: ltY},
		RB:       &effect.PointRestModel{X: rbX, Y: rbY},
	})
	if err != nil {
		panic(err)
	}
	return m
}
```

Drop the `point` import from this test file — `effect.PointRestModel` carries the rectangle.

Create `poisonbomb/poisonbomb_test.go` as the same three tests with
`skill2.NightWalkerStage3PoisonBomb`, wire id `14111006`, rect
`lt(-100,-82)`/`rb(100,83)`, and the same `POISON` / `DiseaseValue: 0`
assertions (FR-6.2). Write the code out in full — do not `//` reference the
flamegear file.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/flamegear/ ./skill/handler/poisonbomb/ -v`
Expected: FAIL — no non-test Go files in either package.

- [ ] **Step 3: Write the handlers**

Create `services/atlas-channel/atlas.com/channel/skill/handler/flamegear/flamegear.go`:

```go
// Package flamegear implements the Blaze Wizard Flame Gear (12111005) cast:
// it places a server-side mist at the caster's feet that poisons every
// monster inside its rectangle until it expires.
package flamegear

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/skill/handler/mistcast"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Flame Gear is ATTACK-delivered, so it registers on the attack-cast
// registry: `GET /api/data/skills/12111005` serves prop 0.51->0.80 on gms
// 72/79/83/84/87/92 and jms 185, and mobCount 8 / damage 124 at gms 95 --
// real attack nodes, not the reader's absent-node defaults (damage 100,
// attackCount 1, mobCount 1). Registering it on the use-skill registry would
// never fire AND would suppress its generic MP cost.
func init() {
	channelhandler.RegisterAttackCast(skill2.BlazeWizardStage3FlameGear, Apply)
}

var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// Apply is the Flame Gear handler installed in the per-skill attack-cast
// registry.
//
// The applied status is POISON with a target-derived magnitude (sent as 0 and
// resolved per monster by atlas-monsters' ResolvePoisonDamage). This is
// WZ-derived, not an analogy with Poison Mist: `monsterStatus` is {} for
// 12111005 on every version served, atlas-monsters implements exactly two DoT
// statuses (POISON and the caster-magnitude VENOM), and the only caster-side
// magnitude candidate -- `dot` -- is 0 on six of the seven versions that bind
// the skill (task-218 design §1.4).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	skillId skill2.Id,
	skillLevel byte,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		skillId skill2.Id,
		skillLevel byte,
		e effect.Model,
	) error {
		return func(
			_ writer.Producer,
			f field.Model,
			characterId uint32,
			skillId skill2.Id,
			skillLevel byte,
			e effect.Model,
		) error {
			return mistcast.Cast(l, ctx, f, characterId, skillId, skillLevel, e,
				mistcast.Params{
					SkillName:  "Flame Gear",
					TargetKind: mistmsg.TargetKindMonster,
					EffectKind: mistmsg.EffectKindDamageOverTime,
					Disease:    "POISON",
					TickMs:     mistcast.PlayerMistTickIntervalMs,
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
```

Create `poisonbomb/poisonbomb.go` identically, with:
- package doc: `// Package poisonbomb implements the Night Walker Poison Bomb (14111006) cast: …`
- `channelhandler.RegisterAttackCast(skill2.NightWalkerStage3PoisonBomb, Apply)`
- the init comment citing `mobCount` 6 and `damage` 104→220 on every version, `prop` 0.51→0.80 pre-v95
- `SkillName: "Poison Bomb"`
- an `Apply` doc comment noting the POISON magnitude is target-derived and sent as 0 (FR-6.2), exactly as Poison Mist does.

- [ ] **Step 4: Blank-import both packages**

In `skill/handler/registrations/registrations.go`, add in alphabetical position:

```go
	_ "atlas-channel/skill/handler/flamegear"    // Blaze Wizard Flame Gear — task-218
	_ "atlas-channel/skill/handler/poisonbomb"   // Night Walker Poison Bomb — task-218
```

A handler package that is not imported never runs its `init()` and is silently absent (FR-7.5).

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/... -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler
git commit -m "feat(task-218): Flame Gear and Poison Bomb cast monster DoT mists"
```

---

### Task 11: Smokescreen handler

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/smokescreen/smokescreen.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/smokescreen/smokescreen_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `channelhandler.Register` (USE_SKILL), `skill2.ShadowerSmokescreen`, `packetmodel.SkillUsageInfo` (`SkillId()`, `SkillLevel()`), `mistcast.*`.
- Produces: `smokescreen.Apply` of type `channelhandler.Handler`.

**Registry evidence (FR-7.2):** 4221006 serves `damage` 100, `attackCount` 1, `mobCount` 1, `prop` 0 on **every** one of the ten live tenants — i.e. every attack node absent, per the reader's default table (`skill/reader.go:197,268,270`). It trips none of the three discriminators, so it is USE_SKILL-delivered. `UseSkill` applies MP consume and the 600 s (360 s at v95) cooldown *before* the handler lookup, so this handler charges nothing itself.

- [ ] **Step 1: Write the failing test**

Create `services/atlas-channel/atlas.com/channel/skill/handler/smokescreen/smokescreen_test.go`:

```go
package smokescreen

import (
	"context"
	"testing"

	"atlas-channel/data/skill/effect"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// FR-7.1/FR-7.2: 4221006 has no attack nodes on any of the ten live tenants,
// so the client delivers it on USE_SKILL and it belongs on the use-skill
// registry -- where UseSkill has already charged MP and the cooldown.
func TestInit_RegistersOnUseSkillRegistryByIdentity(t *testing.T) {
	_, ok := channelhandler.Lookup(skill2.ShadowerSmokescreen)
	require.True(t, ok, "Smokescreen must be registered on the use-skill registry")

	_, wrong := channelhandler.LookupAttackCast(skill2.ShadowerSmokescreen)
	require.False(t, wrong, "Smokescreen must NOT be on the attack-cast registry")
}

func TestApply_EmitsProtectionMistThatNeverTicks(t *testing.T) {
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit := loadCaster, emitCreate
	t.Cleanup(func() { loadCaster, emitCreate = origLoad, origEmit })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) {
		return -40, 200, nil
	}
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}

	e, err := effect.Extract(effect.RestModel{
		Duration: 31000,
		LT:       &effect.PointRestModel{X: -110, Y: -82},
		RB:       &effect.PointRestModel{X: 110, Y: 83},
	})
	require.NoError(t, err)
	f := field.NewBuilder(0, 0, 100000000).Build()
	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(4221006).SetSkillLevel(30).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, info, e))

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, mistmsg.TargetKindCharacter, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindProtection, b.EffectKind)
	// A protection mist has no atlas-maps tick: the shield is evaluated on
	// the channel's damage path. A non-zero interval here would make the
	// tick's PROTECTION arm reachable.
	require.Equal(t, int64(0), b.TickIntervalMs)
	require.Equal(t, "", b.Disease)
	require.Equal(t, int32(0), b.DiseaseValue)
	require.Equal(t, int32(0), b.RecoveryMp)
	// FR-7.4: the WIRE id, not the Identity.
	require.Equal(t, uint32(4221006), b.SourceSkillId)
	require.Equal(t, uint32(30), b.SourceSkillLevel)
	require.Equal(t, int64(31000), b.Duration)
}
```

`packetmodel.NewSkillUsageInfoBuilder()` with `SetSkillId(uint32)` / `SetSkillLevel(byte)` is the real API (`libs/atlas-packet/model/skill_usage_info.go:97-115`); `SkillId()` returns `uint32` and `SkillLevel()` returns `byte`. Drop the `point` import — `effect.PointRestModel` carries the rectangle.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/smokescreen/ -v`
Expected: FAIL — no non-test Go files.

- [ ] **Step 3: Write the handler**

Create `services/atlas-channel/atlas.com/channel/skill/handler/smokescreen/smokescreen.go`:

```go
// Package smokescreen implements the Shadower Smokescreen (4221006) cast: it
// places a server-side smoke cloud at the caster's feet that shields the
// caster and their online party members from damage while they stand inside
// it.
//
// The mist itself never ticks. atlas-maps holds it and expires it; the
// protection is evaluated in atlas-channel on the damage path
// (socket/handler/character_damage_smoke.go), which is where the client
// evaluates it too -- CUserLocal::SetDamaged consults
// CAffectedAreaPool::IsSmokeAreaByPoint and, on a hit, jumps straight to the
// function epilogue before the miss roll, Power Guard, Meso Guard, Achilles
// and Magic Guard, sending no damage packet at all.
package smokescreen

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/skill/handler/mistcast"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// Smokescreen is USE_SKILL-delivered, so it registers on the use-skill
// registry: `GET /api/data/skills/4221006` serves damage 100, attackCount 1,
// mobCount 1 and prop 0 on all ten live tenants -- i.e. every attack node is
// ABSENT (those are the reader's defaults for a missing node,
// atlas-data skill/reader.go:197,268,270), not present-and-equal-to-default.
// UseSkill charges its MP consume and its 600s (360s at v95) cooldown before
// the handler lookup, so this handler charges nothing itself.
func init() {
	channelhandler.Register(skill2.ShadowerSmokescreen, Apply)
}

var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// Apply is the Smokescreen handler installed in the per-skill use-skill
// registry.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	info packetmodel.SkillUsageInfo,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		info packetmodel.SkillUsageInfo,
		e effect.Model,
	) error {
		return func(
			_ writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
			e effect.Model,
		) error {
			return mistcast.Cast(l, ctx, f, characterId, skill2.Id(info.SkillId()), info.SkillLevel(), e,
				mistcast.Params{
					SkillName:  "Smokescreen",
					TargetKind: mistmsg.TargetKindCharacter,
					EffectKind: mistmsg.EffectKindProtection,
					// A protection mist has no per-tick effect: atlas-maps
					// only holds and expires it. TickMs 0 makes
					// Mist.ShouldTick false, so the tick returns before the
					// effect-kind switch is ever reached.
					TickMs: 0,
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
```

Match `info.SkillId()`'s and `info.SkillLevel()`'s real return types to the conversions above — adjust the casts, not the contract.

- [ ] **Step 4: Blank-import the package**

Add to `registrations.go` in alphabetical position:

```go
	_ "atlas-channel/skill/handler/smokescreen"  // Shadower Smokescreen — task-218
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/... -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler
git commit -m "feat(task-218): Smokescreen casts a protection mist"
```

---

### Task 12: Recovery Aura handler and its party snapshot

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/recoveryaura/recoveryaura.go`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/recoveryaura/recoveryaura_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go`

**Interfaces:**
- Consumes: `channelhandler.Register`, `skill2.EvanStage8RecoveryAura` (bound to a wire id only after Task 2), `party.NewProcessor(l, ctx).GetByMemberId(id)` → `Model.Members()` → `MemberModel.Id()/Online()`, `effect.Model.X()`, `mistcast.*`.
- Produces: `recoveryaura.Apply` of type `channelhandler.Handler`; package var `loadPartyMemberIds`.

**Magnitude (FR-1.2):** `x`, already exposed as `effect.Model.X()` and read unscaled at `services/atlas-data/atlas.com/data/skill/reader.go:266`. 38 at L1 → 80 at L15. MP only — `hp`/`mp`/`hpR`/`mpR` are all 0 at every level on every version, and the served description names MP explicitly. No `atlas-data` change (`design.md` §1.3).

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/skill/handler/recoveryaura/recoveryaura_test.go`:

```go
package recoveryaura

import (
	"context"
	"testing"

	"atlas-channel/data/skill/effect"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/skill/handler/mistcast"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func TestInit_RegistersOnUseSkillRegistryByIdentity(t *testing.T) {
	_, ok := channelhandler.Lookup(skill2.EvanStage8RecoveryAura)
	require.True(t, ok, "Recovery Aura must be registered on the use-skill registry")

	_, wrong := channelhandler.LookupAttackCast(skill2.EvanStage8RecoveryAura)
	require.False(t, wrong, "Recovery Aura must NOT be on the attack-cast registry")
}

// run drives Apply with recording seams and a stubbed party snapshot.
func run(t *testing.T, x int16, party []uint32) []mistmsg.CreateCommandBody {
	t.Helper()
	l, _ := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit, origParty := loadCaster, emitCreate, loadPartyMemberIds
	t.Cleanup(func() { loadCaster, emitCreate, loadPartyMemberIds = origLoad, origEmit, origParty })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) { return 10, 20, nil }
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}
	loadPartyMemberIds = func(logrus.FieldLogger, context.Context, uint32) []uint32 { return party }

	e := stubEffect(30000, x)
	f := field.NewBuilder(0, 0, 100000000).Build()
	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(22161003).SetSkillLevel(1).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, info, e))
	return emitted
}

// stubEffect hydrates an effect.Model with Recovery Aura's real rectangle
// (lt(-200,-125) rb(200,30), identical on every version that binds it) and
// the given `x` magnitude. effect.Model has unexported fields and no
// exported builder; Extract is the supported construction path.
func stubEffect(duration int32, x int16) effect.Model {
	m, err := effect.Extract(effect.RestModel{
		Duration: duration,
		X:        x,
		LT:       &effect.PointRestModel{X: -200, Y: -125},
		RB:       &effect.PointRestModel{X: 200, Y: 30},
	})
	if err != nil {
		panic(err)
	}
	return m
}

// FR-1.2/FR-5.1: the per-tick magnitude is the WZ `x` node (38 at L1, 80 at
// L15), not a constant, and it travels in RecoveryMp -- never overloaded onto
// DiseaseValue.
func TestApply_MagnitudeIsTheWzXNode(t *testing.T) {
	emitted := run(t, 38, []uint32{1001, 1002})

	require.Len(t, emitted, 1)
	b := emitted[0]
	require.Equal(t, mistmsg.TargetKindCharacter, b.TargetKind)
	require.Equal(t, mistmsg.EffectKindRecovery, b.EffectKind)
	require.Equal(t, int32(38), b.RecoveryMp)
	require.Equal(t, int32(0), b.DiseaseValue)
	require.Equal(t, "", b.Disease)
	require.Equal(t, mistcast.PlayerMistTickIntervalMs, b.TickIntervalMs)
	require.Equal(t, uint32(22161003), b.SourceSkillId)
}

// FR-5.2: the party snapshot travels on the command; atlas-maps has no party
// client and heals only ids in this set.
func TestApply_CarriesThePartySnapshot(t *testing.T) {
	emitted := run(t, 38, []uint32{1001, 1002, 1003})
	require.Equal(t, []uint32{1001, 1002, 1003}, emitted[0].PartyMemberIds)
}

// A magnitude of 0 would create a mist that heals nothing every 3s for 30s.
// Reject it rather than emit a no-op cloud.
func TestApply_ZeroMagnitude_Rejected(t *testing.T) {
	l, hook := test.NewNullLogger()
	var emitted []mistmsg.CreateCommandBody

	origLoad, origEmit, origParty := loadCaster, emitCreate, loadPartyMemberIds
	t.Cleanup(func() { loadCaster, emitCreate, loadPartyMemberIds = origLoad, origEmit, origParty })
	loadCaster = func(logrus.FieldLogger, context.Context, uint32) (int16, int16, error) { return 0, 0, nil }
	emitCreate = func(_ logrus.FieldLogger, _ context.Context, body mistmsg.CreateCommandBody) error {
		emitted = append(emitted, body)
		return nil
	}
	loadPartyMemberIds = func(logrus.FieldLogger, context.Context, uint32) []uint32 { return []uint32{1001} }

	e := stubEffect(30000, 0)
	f := field.NewBuilder(0, 0, 100000000).Build()
	info := packetmodel.NewSkillUsageInfoBuilder().SetSkillId(22161003).SetSkillLevel(1).Build()

	require.NoError(t, Apply(l)(context.Background())(nil, f, 1001, info, e))
	require.Empty(t, emitted)
	require.Contains(t, hook.LastEntry().Message, "no recovery magnitude")
}

// A soloing caster still gets their own aura: the snapshot always contains
// the caster, even when the party lookup returns nothing.
func TestPartyMemberIdsOrSelf_AlwaysIncludesCaster(t *testing.T) {
	require.Equal(t, []uint32{1001}, withCaster(nil, 1001))
	require.Equal(t, []uint32{1002, 1001}, withCaster([]uint32{1002}, 1001))
	require.Equal(t, []uint32{1001, 1002}, withCaster([]uint32{1001, 1002}, 1001))
}
```

Drop the `point` import from this test file — `effect.PointRestModel` carries the rectangle.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/recoveryaura/ -v`
Expected: FAIL — no non-test Go files.

- [ ] **Step 3: Write the handler**

Create `services/atlas-channel/atlas.com/channel/skill/handler/recoveryaura/recoveryaura.go`:

```go
// Package recoveryaura implements the Evan Recovery Aura (22161003) cast: it
// places a server-side aura at the caster's feet that periodically restores
// MP to the caster's party members standing inside it.
package recoveryaura

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/party"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/skill/handler/mistcast"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// Recovery Aura is USE_SKILL-delivered, so it registers on the use-skill
// registry: `GET /api/data/skills/22161003` serves damage 100, attackCount 1,
// mobCount 1 and prop 0 on every version that binds it (gms 84/87/92/95 and
// jms 185) -- i.e. every attack node is ABSENT. UseSkill charges its
// HPConsume (18->34), its equal MPConsume, and its 60s cooldown before the
// handler lookup, so this handler charges nothing itself.
func init() {
	channelhandler.Register(skill2.EvanStage8RecoveryAura, Apply)
}

var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// loadPartyMemberIds returns the caster's online party member ids. Package
// var so tests can stub it.
//
// This is a CAST-TIME snapshot, carried on the CREATE command, because
// atlas-maps has no party client and giving it one would add a service edge
// for a rule nothing client-side evaluates. The cost is a staleness window
// bounded by the aura's fixed 30s lifetime: someone who joins the party
// mid-aura is not healed, someone who leaves still is.
//
// Smokescreen deliberately does NOT work this way -- the client independently
// evaluates live party membership for smoke at hit time
// (CAffectedAreaPool::IsSmokeAreaByPoint), so a server snapshot there would
// visibly disagree with the player's own screen.
var loadPartyMemberIds = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) []uint32 {
	p, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
	if err != nil {
		// No party is the common case for a soloing caster, not an error
		// worth an error-level line; the caster still gets their own aura.
		l.WithError(err).Debugf("Recovery Aura: no party for character [%d]; scoping the aura to the caster.", characterId)
		return nil
	}
	ids := make([]uint32, 0, len(p.Members()))
	for _, m := range p.Members() {
		if !m.Online() {
			continue
		}
		ids = append(ids, m.Id())
	}
	return ids
}

// withCaster guarantees the caster is in the snapshot, so a soloing Evan --
// or one whose party lookup failed -- still benefits from their own aura.
func withCaster(ids []uint32, characterId uint32) []uint32 {
	for _, id := range ids {
		if id == characterId {
			return ids
		}
	}
	return append(ids, characterId)
}

// Apply is the Recovery Aura handler installed in the per-skill use-skill
// registry.
//
// The per-tick magnitude is the WZ `x` node (38 at L1 rising to 80 at L15),
// restored as MP: the skill's served description names MP explicitly, and
// `hp`, `mp`, `hpR` and `mpR` are all 0 at every level on every version that
// binds it (task-218 design §1.3). `x` is treated as an absolute MP amount,
// consistent with every other `x` consumer in this service.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	info packetmodel.SkillUsageInfo,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		info packetmodel.SkillUsageInfo,
		e effect.Model,
	) error {
		return func(
			_ writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
			e effect.Model,
		) error {
			if e.X() <= 0 {
				l.Warnf("Recovery Aura: rejected cast by [%d] — no recovery magnitude (WZ x = %d).", characterId, e.X())
				return nil
			}
			return mistcast.Cast(l, ctx, f, characterId, skill2.Id(info.SkillId()), info.SkillLevel(), e,
				mistcast.Params{
					SkillName:      "Recovery Aura",
					TargetKind:     mistmsg.TargetKindCharacter,
					EffectKind:     mistmsg.EffectKindRecovery,
					TickMs:         mistcast.PlayerMistTickIntervalMs,
					RecoveryMp:     int32(e.X()),
					PartyMemberIds: withCaster(loadPartyMemberIds(l, ctx, characterId), characterId),
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
```

Over the fixed 30 000 ms lifetime at a 3000 ms cadence that is 10 ticks — 380 MP at L1, 800 MP at L15.

- [ ] **Step 4: Blank-import the package**

Add to `registrations.go` in alphabetical position:

```go
	_ "atlas-channel/skill/handler/recoveryaura" // Evan Recovery Aura — task-218
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./skill/... -v && go build ./...`
Expected: PASS, build clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler
git commit -m "feat(task-218): Recovery Aura casts a party-scoped MP recovery mist"
```

---

### Task 13: Channel-side protection-mist registry

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/mist/protection.go`
- Create: `services/atlas-channel/atlas.com/channel/mist/protection_test.go`

**Interfaces:**
- Produces (consumed by Tasks 14, 15):
  - `mist.Protection` — immutable, private fields + getters: `Id() uuid.UUID`, `Field() field.Model`, `OwnerId() uint32`, `Contains(x, y int16) bool`, `ExpiresAt() time.Time`, `Expired(now time.Time) bool`
  - `mist.NewProtectionBuilder(id uuid.UUID, f field.Model) *ProtectionBuilder` with `SetOwnerId(uint32)`, `SetRect(minX, minY, maxX, maxY int16)`, `SetExpiresAt(time.Time)`, `Build() Protection`
  - `mist.GetProtectionRegistry() *ProtectionRegistry`, `mist.NewTestProtectionRegistry() *ProtectionRegistry`
  - `(*ProtectionRegistry).Add(t tenant.Model, p Protection)`, `.Remove(t tenant.Model, id uuid.UUID)`, `.Covering(t tenant.Model, f field.Model, x, y int16, now time.Time) []Protection`

**Rect convention:** absolute world coordinates with **inclusive** edges, matching atlas-maps' `Mist.Rect`/`Mist.Contains` (`mist/model.go:265-274`) and the atlas-monsters in-rect endpoint. The duplication is deliberate (the alternative is a per-hit REST round trip); the convention is asserted on both sides.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/mist/protection_test.go`:

```go
package mist_test

import (
	"testing"
	"time"

	"atlas-channel/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tn
}

// protectionAt builds a live protection covering (0,0)..(200,200).
func protectionAt(f field.Model, ownerId uint32, ttl time.Duration) mist.Protection {
	return mist.NewProtectionBuilder(uuid.New(), f).
		SetOwnerId(ownerId).
		SetRect(0, 0, 200, 200).
		SetExpiresAt(time.Now().Add(ttl)).
		Build()
}

func TestCovering_ReturnsProtectionContainingThePoint(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	p := protectionAt(f, 1001, time.Minute)
	r.Add(tn, p)

	require.Len(t, r.Covering(tn, f, 100, 100, time.Now()), 1)
	// Inclusive edges, matching atlas-maps' Mist.Contains.
	require.Len(t, r.Covering(tn, f, 0, 0, time.Now()), 1)
	require.Len(t, r.Covering(tn, f, 200, 200, time.Now()), 1)
	require.Empty(t, r.Covering(tn, f, 201, 100, time.Now()))
	require.Empty(t, r.Covering(tn, f, 100, -1, time.Now()))
}

func TestCovering_IgnoresOtherFieldsAndTenants(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	other := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	elsewhere := field.NewBuilder(0, 0, 100000001).Build()
	r.Add(tn, protectionAt(f, 1001, time.Minute))

	require.Empty(t, r.Covering(tn, elsewhere, 100, 100, time.Now()))
	require.Empty(t, r.Covering(other, f, 100, 100, time.Now()))
}

// A dropped MIST_DESTROYED must not leave a permanently protective
// rectangle: expiry is evaluated on read.
func TestCovering_TreatsExpiredAsAbsent(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	r.Add(tn, protectionAt(f, 1001, -time.Second))

	require.Empty(t, r.Covering(tn, f, 100, 100, time.Now()))
}

func TestRemove_DropsTheProtection(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	p := protectionAt(f, 1001, time.Minute)
	r.Add(tn, p)
	require.Len(t, r.Covering(tn, f, 100, 100, time.Now()), 1)

	r.Remove(tn, p.Id())

	require.Empty(t, r.Covering(tn, f, 100, 100, time.Now()))
}

// Add prunes expired entries lazily, so a channel that never sees a
// MIST_DESTROYED does not accumulate them for the process's lifetime.
func TestAdd_PrunesExpiredEntries(t *testing.T) {
	r := mist.NewTestProtectionRegistry()
	tn := testTenant(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	r.Add(tn, protectionAt(f, 1001, -time.Second))
	r.Add(tn, protectionAt(f, 1002, time.Minute))

	require.Equal(t, 1, r.Len(tn))
}

func TestGetProtectionRegistry_IsASingleton(t *testing.T) {
	require.Same(t, mist.GetProtectionRegistry(), mist.GetProtectionRegistry())
}
```

Use whatever tenant test constructor this service's other tests already use (`tenant.Create(...)` or a package helper) rather than the shape above if it differs.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./mist/ -v`
Expected: FAIL — `NewProtectionBuilder`, `NewTestProtectionRegistry` undefined.

- [ ] **Step 3: Write the implementation**

Create `services/atlas-channel/atlas.com/channel/mist/protection.go`:

```go
package mist

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Protection is a live protection (Smokescreen) mist as this channel knows
// it: enough to answer "is this character standing in one, and does it belong
// to them or their party?" on the damage path.
//
// The channel keeps its own copy rather than querying atlas-maps because the
// alternative is a synchronous REST round trip on the most latency-sensitive
// path in the service. The cost is a restart gap: the mist consumer starts at
// kafka.LastOffset, so a channel that restarts mid-mist never learns about
// mists created before it came up. That is not a regression -- the same
// restart already loses the AffectedAreaCreated broadcast, so those mists are
// invisible to every client on that channel too -- and it is bounded by the
// longest Smokescreen lifetime (60s at level 30).
type Protection struct {
	id        uuid.UUID
	f         field.Model
	ownerId   uint32
	minX      int16
	minY      int16
	maxX      int16
	maxY      int16
	expiresAt time.Time
}

// Id returns the mist id this protection was created from.
func (p Protection) Id() uuid.UUID { return p.id }

// Field returns the field the protection covers.
func (p Protection) Field() field.Model { return p.f }

// OwnerId returns the casting character. The client's smoke lookup accepts an
// area only if its dwOwnerID is the local character or one of their online
// party members (CAffectedAreaPool::IsSmokeAreaByPoint, v95 @0x434f40), so
// the owner is what the party check is evaluated against.
func (p Protection) OwnerId() uint32 { return p.ownerId }

// ExpiresAt returns the absolute expiry.
func (p Protection) ExpiresAt() time.Time { return p.expiresAt }

// Expired reports whether the protection is past its lifetime as of now.
func (p Protection) Expired(now time.Time) bool { return now.After(p.expiresAt) }

// Contains reports whether the world coordinates fall inside the protection's
// axis-aligned bounding box. Edges are INCLUSIVE, matching atlas-maps'
// Mist.Contains and the atlas-monsters in-rect endpoint -- the rect test
// exists on both sides and the two conventions must not drift.
func (p Protection) Contains(x, y int16) bool {
	return x >= p.minX && x <= p.maxX && y >= p.minY && y <= p.maxY
}

// ProtectionBuilder constructs a Protection via fluent setters.
type ProtectionBuilder struct {
	p Protection
}

// NewProtectionBuilder starts a Protection anchored to a mist id and field.
func NewProtectionBuilder(id uuid.UUID, f field.Model) *ProtectionBuilder {
	return &ProtectionBuilder{p: Protection{id: id, f: f}}
}

func (b *ProtectionBuilder) SetOwnerId(v uint32) *ProtectionBuilder {
	b.p.ownerId = v
	return b
}

// SetRect sets the ABSOLUTE world-coordinate bounding box (origin already
// added to the lt/rb offsets).
func (b *ProtectionBuilder) SetRect(minX, minY, maxX, maxY int16) *ProtectionBuilder {
	b.p.minX, b.p.minY, b.p.maxX, b.p.maxY = minX, minY, maxX, maxY
	return b
}

func (b *ProtectionBuilder) SetExpiresAt(v time.Time) *ProtectionBuilder {
	b.p.expiresAt = v
	return b
}

func (b *ProtectionBuilder) Build() Protection { return b.p }

// ProtectionRegistry is a tenant-scoped, in-memory index of live protection
// mists. Safe for concurrent use: the damage path reads it on every hit.
type ProtectionRegistry struct {
	mu        sync.RWMutex
	perTenant map[string]map[uuid.UUID]Protection
}

var (
	protectionRegistryOnce sync.Once
	protectionRegistry     *ProtectionRegistry
)

// GetProtectionRegistry returns the process-wide singleton, lazily built.
func GetProtectionRegistry() *ProtectionRegistry {
	protectionRegistryOnce.Do(func() {
		protectionRegistry = &ProtectionRegistry{perTenant: map[string]map[uuid.UUID]Protection{}}
	})
	return protectionRegistry
}

// NewTestProtectionRegistry returns a fresh, isolated registry so tests do
// not leak state through the singleton. Not used in production paths.
func NewTestProtectionRegistry() *ProtectionRegistry {
	return &ProtectionRegistry{perTenant: map[string]map[uuid.UUID]Protection{}}
}

// Add inserts p and lazily prunes entries that have already expired, so a
// dropped MIST_DESTROYED cannot accumulate stale rectangles.
func (r *ProtectionRegistry) Add(t tenant.Model, p Protection) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := t.Id().String()
	b, ok := r.perTenant[key]
	if !ok {
		b = map[uuid.UUID]Protection{}
		r.perTenant[key] = b
	}
	now := time.Now()
	for id, existing := range b {
		if existing.Expired(now) {
			delete(b, id)
		}
	}
	b[p.Id()] = p
}

// Remove drops the protection with the given mist id. No-op if absent.
func (r *ProtectionRegistry) Remove(t tenant.Model, id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := t.Id().String()
	b, ok := r.perTenant[key]
	if !ok {
		return
	}
	delete(b, id)
	if len(b) == 0 {
		delete(r.perTenant, key)
	}
}

// Covering returns every live protection on f that contains (x, y). Expired
// entries are treated as absent regardless of whether they have been pruned:
// a missed MIST_DESTROYED must degrade to "no protection", never to a
// permanently invulnerable rectangle.
func (r *ProtectionRegistry) Covering(t tenant.Model, f field.Model, x, y int16, now time.Time) []Protection {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.perTenant[t.Id().String()]
	if !ok {
		return nil
	}
	var out []Protection
	for _, p := range b {
		if p.Expired(now) || !p.Field().Equals(f) || !p.Contains(x, y) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// Len reports how many protections the tenant currently holds. Exposed for
// tests asserting the pruning behaviour.
func (r *ProtectionRegistry) Len(t tenant.Model) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.perTenant[t.Id().String()])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./mist/ -v`
Expected: PASS (7 tests).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/mist
git commit -m "feat(task-218): channel-local protection mist registry"
```

---

### Task 14: Populate the protection registry from `EVENT_TOPIC_MIST`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go:89-127`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer_test.go`

**Interfaces:**
- Consumes: `mistmsg.CreatedBody.EffectKind` (Task 7), `mist.NewProtectionBuilder`, `mist.GetProtectionRegistry` (Task 13).
- Produces: `handleMistCreated` also registers `EffectKind == PROTECTION` mists; `handleMistDestroyed` also evicts them. The registry the handlers use is a package var (`protectionRegistry`) so tests can point it at `mist.NewTestProtectionRegistry()`.

The broadcast behaviour is unchanged — every mist still becomes an `AffectedAreaCreated`/`AffectedAreaRemoved`; the registry work is additive.

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer_test.go` (reuse the file's existing broadcaster-stub and tenant-context helpers):

```go
// A PROTECTION mist must land in the channel's registry with its ABSOLUTE
// rect (origin + offsets) so the damage path can test a character's position
// against it directly.
func TestHandleMistCreated_RegistersProtectionMistWithAbsoluteRect(t *testing.T) {
	reg := mist.NewTestProtectionRegistry()
	orig := protectionRegistry
	t.Cleanup(func() { protectionRegistry = orig })
	protectionRegistry = reg

	ctx, tn := tenantContext(t)
	mistId := uuid.New()
	e := mist2.Event[mist2.CreatedBody]{
		WorldId: 0, ChannelId: 0, MapId: 100000000, MistId: mistId,
		Type: mist2.EventTypeCreated,
		Body: mist2.CreatedBody{
			OwnerType: "CHARACTER", OwnerId: 1001,
			EffectKind: mist2.EffectKindProtection,
			Type:       2,
			OriginX:    500, OriginY: 300,
			LtX: -110, LtY: -82, RbX: 110, RbY: 83,
			Duration: 31000,
		},
	}

	handleMistCreated(testServer(t), testWriterProducer(t))(testLogger(t), ctx, e)

	f := field.NewBuilder(0, 0, 100000000).Build()
	// origin 500,300 + lt(-110,-82)..rb(110,83) => (390,218)..(610,383)
	require.Len(t, reg.Covering(tn, f, 500, 300, time.Now()), 1)
	require.Len(t, reg.Covering(tn, f, 390, 218, time.Now()), 1)
	require.Empty(t, reg.Covering(tn, f, 389, 300, time.Now()))
	require.Equal(t, uint32(1001), reg.Covering(tn, f, 500, 300, time.Now())[0].OwnerId())
}

// Non-protection mists must NOT enter the registry -- a Poison Mist that
// shielded its caster would be a silent invulnerability.
func TestHandleMistCreated_IgnoresNonProtectionMists(t *testing.T) {
	reg := mist.NewTestProtectionRegistry()
	orig := protectionRegistry
	t.Cleanup(func() { protectionRegistry = orig })
	protectionRegistry = reg

	ctx, tn := tenantContext(t)
	e := mist2.Event[mist2.CreatedBody]{
		WorldId: 0, ChannelId: 0, MapId: 100000000, MistId: uuid.New(),
		Type: mist2.EventTypeCreated,
		Body: mist2.CreatedBody{
			OwnerType: "CHARACTER", OwnerId: 1001,
			EffectKind: mist2.EffectKindDamageOverTime,
			OriginX:    500, OriginY: 300,
			LtX: -110, LtY: -82, RbX: 110, RbY: 83,
			Duration: 40000,
		},
	}

	handleMistCreated(testServer(t), testWriterProducer(t))(testLogger(t), ctx, e)

	f := field.NewBuilder(0, 0, 100000000).Build()
	require.Empty(t, reg.Covering(tn, f, 500, 300, time.Now()))
}

// FR-4.3: protection ends on expiry AND on cancellation.
func TestHandleMistDestroyed_EvictsTheProtection(t *testing.T) {
	reg := mist.NewTestProtectionRegistry()
	orig := protectionRegistry
	t.Cleanup(func() { protectionRegistry = orig })
	protectionRegistry = reg

	ctx, tn := tenantContext(t)
	f := field.NewBuilder(0, 0, 100000000).Build()
	mistId := uuid.New()
	reg.Add(tn, mist.NewProtectionBuilder(mistId, f).
		SetOwnerId(1001).SetRect(390, 218, 610, 383).
		SetExpiresAt(time.Now().Add(time.Minute)).Build())

	handleMistDestroyed(testServer(t), testWriterProducer(t))(testLogger(t), ctx,
		mist2.Event[mist2.DestroyedBody]{
			WorldId: 0, ChannelId: 0, MapId: 100000000, MistId: mistId,
			Type: mist2.EventTypeDestroyed,
			Body: mist2.DestroyedBody{Reason: mist2.ReasonCancelled},
		})

	require.Empty(t, reg.Covering(tn, f, 500, 300, time.Now()))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/mist/ -v`
Expected: FAIL — `protectionRegistry` undefined.

- [ ] **Step 3: Write the implementation**

In `kafka/consumer/mist/consumer.go`, add next to the `mistPhase` const:

```go
// protectionRegistry is the channel-local index of live PROTECTION
// (Smokescreen) mists, consulted by the damage path. Held as a package var so
// tests can point it at an isolated registry.
//
// A protection mist is recognised by its EffectKind rather than by the
// client-facing `Type` (nType 2): nType is a render detail, and inferring the
// domain concept from it would couple channel logic to the client's value
// table -- exactly what AffectedAreaTypeFor's doc comment exists to prevent.
var protectionRegistry = mist.GetProtectionRegistry()
```

At the end of `handleMistCreated`, after the broadcast:

```go
			if e.Body.EffectKind == mist2.EffectKindProtection {
				protectionRegistry.Add(tenant.MustFromContext(ctx),
					mist.NewProtectionBuilder(e.MistId, f).
						SetOwnerId(e.Body.OwnerId).
						// Absolute world rect: the event carries the origin
						// and the lt/rb OFFSETS, and the damage path tests a
						// character's absolute position.
						SetRect(e.Body.OriginX+e.Body.LtX, e.Body.OriginY+e.Body.LtY,
							e.Body.OriginX+e.Body.RbX, e.Body.OriginY+e.Body.RbY).
						SetExpiresAt(time.Now().Add(time.Duration(e.Body.Duration) * time.Millisecond)).
						Build())
			}
```

At the end of `handleMistDestroyed`, after the broadcast:

```go
			// Unconditional: removing an id that was never a protection mist
			// is a no-op, and this way a PROTECTION mist can never survive
			// its own destruction because of a kind check that drifted.
			protectionRegistry.Remove(tenant.MustFromContext(ctx), e.MistId)
```

Add `"atlas-channel/mist"` and `"time"` to the imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./kafka/consumer/mist/ -v`
Expected: PASS — including the pre-existing broadcast tests, unchanged.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/mist
git commit -m "feat(task-218): track protection mists in the channel mist consumer"
```

---

### Task 15: Smokescreen damage short-circuit

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_smoke.go`
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_smoke_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go:34-44,83-97,168-198`

**Interfaces:**
- Consumes: `mist.Protection`, `mist.GetProtectionRegistry` (Task 13); `party.NewProcessor(l, ctx).GetByMemberId(id)`.
- Produces:
  - `shieldedBySmoke(covering []mist.Protection, characterId uint32, partyIds func() []uint32) bool` — pure predicate
  - `newSmokeCheck(l logrus.FieldLogger, ctx context.Context, t tenant.Model) func(f field.Model, characterId uint32, x, y int16) bool`
  - `damageMitigationDeps.inProtectiveMist func(f field.Model, characterId uint32, x, y int16) bool`

**Client behaviour this mirrors (`design.md` §2.2–2.3):** `CUserLocal::SetDamaged` calls `CAffectedAreaPool::IsSmokeAreaByPoint` with the online party array filled immediately before by `CWvsContext::GetOnlinePartyMemberID`; a positive result jumps straight to the epilogue (`SetDamaged+0x1ef` → `loc_93651F`) **before** the miss roll, Power Guard, Meso Guard, Achilles, and Magic Guard, and before the damage packet is built. So this is a short-circuit, not another term in the mitigation chain — modelling it as a percentage reduction inside `computeMitigation` would diverge from the client, and would let reflect and Meso Guard amounts be computed from damage the shield zeroed (FR-4.5).

An honest client sends nothing while in smoke, so this server-side check exists to stop a **crafted** client claiming damage it did not take, and to cover server-initiated damage. That is FR-4.2, and it is the whole point of the path.

- [ ] **Step 1: Write the failing tests**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_smoke_test.go`:

```go
package handler

import (
	"testing"
	"time"

	"atlas-channel/mist"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

func smokeAt(f field.Model, ownerId uint32) mist.Protection {
	return mist.NewProtectionBuilder(uuid.New(), f).
		SetOwnerId(ownerId).
		SetRect(0, 0, 200, 200).
		SetExpiresAt(time.Now().Add(time.Minute)).
		Build()
}

// FR-4.6: the client accepts a smoke area only if its owner is the local
// character or one of their ONLINE party members
// (CAffectedAreaPool::IsSmokeAreaByPoint, v95 @0x434f40). The server must
// match, or a non-party player renders unharmed while the server kills them.
func TestShieldedBySmoke(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000000).Build()
	party := func() []uint32 { return []uint32{2001, 2002} }
	noParty := func() []uint32 { return nil }

	tests := []struct {
		name        string
		covering    []mist.Protection
		characterId uint32
		partyIds    func() []uint32
		want        bool
	}{
		{"own mist shields the caster", []mist.Protection{smokeAt(f, 1001)}, 1001, noParty, true},
		{"party member's mist shields", []mist.Protection{smokeAt(f, 2001)}, 1001, party, true},
		{"non-party mist does not shield", []mist.Protection{smokeAt(f, 3001)}, 1001, party, false},
		{"no covering mist does not shield", nil, 1001, party, false},
		{"one qualifying mist among several is enough", []mist.Protection{smokeAt(f, 3001), smokeAt(f, 2002)}, 1001, party, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, shieldedBySmoke(tc.covering, tc.characterId, tc.partyIds))
		})
	}
}

// The party lookup is a REST call; it must not fire when no mist could
// possibly qualify, i.e. on every ordinary hit.
func TestShieldedBySmoke_DoesNotResolvePartyWhenUnnecessary(t *testing.T) {
	f := field.NewBuilder(0, 0, 100000000).Build()
	calls := 0
	party := func() []uint32 { calls++; return []uint32{2001} }

	require.False(t, shieldedBySmoke(nil, 1001, party))
	require.Equal(t, 0, calls, "no covering mists: party must not be resolved")

	require.True(t, shieldedBySmoke([]mist.Protection{smokeAt(f, 1001)}, 1001, party))
	require.Equal(t, 0, calls, "own mist matches first: party must not be resolved")

	require.False(t, shieldedBySmoke([]mist.Protection{smokeAt(f, 3001)}, 1001, party))
	require.Equal(t, 1, calls, "foreign mist: party resolved exactly once")
}
```

Append to `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_test.go` (reuse the file's existing `damageMitigationDeps` fake builder and `packetmodel.NewDamageTakenInfoBuilder`-equivalent helpers):

```go
// FR-4.1/FR-4.5: a shielded character takes zero HP damage, and NOTHING else
// on the mitigation chain runs -- no reflect, no meso spend, no MP loss --
// matching the client's own short-circuit before Power Guard/Meso Guard.
func TestProcessDamageTaken_ProtectiveMist_ZeroesEverything(t *testing.T) {
	var (
		hpCalls     int
		mpCalls     int
		mesoCalls   int
		reflectHits int
		buffLookups int
	)
	deps := newFakeDeps()
	deps.getBuffs = func(uint32) ([]buff.Model, error) { buffLookups++; return nil, nil }
	deps.changeHP = func(field.Model, uint32, int16) error { hpCalls++; return nil }
	deps.changeMP = func(field.Model, uint32, int16) error { mpCalls++; return nil }
	deps.requestChangeMeso = func(field.Model, uint32, uint32, string, int32) error { mesoCalls++; return nil }
	deps.damageMonster = func(field.Model, uint32, uint32, []uint32, byte) error { reflectHits++; return nil }
	deps.inProtectiveMist = func(field.Model, uint32, int16, int16) bool { return true }

	processDamageTaken(testLogger(t), testTenant(t), testField(t), damagedFor(500), characterAt(1001, 100, 100), deps)

	require.Zero(t, hpCalls)
	require.Zero(t, mpCalls)
	require.Zero(t, mesoCalls)
	require.Zero(t, reflectHits)
	require.Zero(t, buffLookups, "the shield short-circuits before the mitigation chain")
}

// FR-4.3: a character who leaves the rectangle takes full damage. The check
// reads the character's CURRENT position, so the worst-case lag is one
// damage event -- there is no tick quantisation.
func TestProcessDamageTaken_OutsideTheMist_TakesFullDamage(t *testing.T) {
	var applied int16
	deps := newFakeDeps()
	deps.changeHP = func(_ field.Model, _ uint32, amount int16) error { applied = amount; return nil }
	deps.inProtectiveMist = func(field.Model, uint32, int16, int16) bool { return false }

	processDamageTaken(testLogger(t), testTenant(t), testField(t), damagedFor(500), characterAt(1001, 900, 900), deps)

	require.Equal(t, int16(-500), applied)
}
```

Adapt the helper names to whatever `character_damage_test.go` already provides; do not add a second set of fakes.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run 'Smoke|ProtectiveMist|OutsideTheMist' -v`
Expected: FAIL — `shieldedBySmoke` undefined; `damageMitigationDeps` has no `inProtectiveMist`.

- [ ] **Step 3: Write the predicate and its wiring**

Create `services/atlas-channel/atlas.com/channel/socket/handler/character_damage_smoke.go`:

```go
package handler

import (
	"atlas-channel/mist"
	"atlas-channel/party"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// shieldedBySmoke reports whether any of the protection mists covering the
// character's position belongs to the character or to one of their online
// party members -- the exact conjunction the client evaluates in
// CAffectedAreaPool::IsSmokeAreaByPoint (v95 @0x434f40): nType == 2, started,
// same phase, owner in the party array or the local character, and the point
// inside the rect. The first four are decided when the protection is
// registered; this is the ownership half.
//
// partyIds is a thunk so the party REST call is made only when a covering
// mist is owned by someone else -- on an ordinary hit there is nothing
// covering the character and no lookup happens at all.
func shieldedBySmoke(covering []mist.Protection, characterId uint32, partyIds func() []uint32) bool {
	if len(covering) == 0 {
		return false
	}
	for _, p := range covering {
		if p.OwnerId() == characterId {
			return true
		}
	}
	ids := partyIds()
	if len(ids) == 0 {
		return false
	}
	inParty := make(map[uint32]bool, len(ids))
	for _, id := range ids {
		inParty[id] = true
	}
	for _, p := range covering {
		if inParty[p.OwnerId()] {
			return true
		}
	}
	return false
}

// newSmokeCheck builds the production inProtectiveMist dependency: the
// channel-local protection registry plus a lazily-resolved online party.
//
// Party membership is resolved at HIT time, not snapshot at cast time,
// because the client resolves it at hit time too (the array passed to
// IsSmokeAreaByPoint is filled immediately before the call by
// CWvsContext::GetOnlinePartyMemberID, v95 @0x93455c). A server snapshot
// would visibly disagree with the player's own screen.
func newSmokeCheck(l logrus.FieldLogger, ctx context.Context, t tenant.Model) func(f field.Model, characterId uint32, x, y int16) bool {
	return func(f field.Model, characterId uint32, x, y int16) bool {
		covering := mist.GetProtectionRegistry().Covering(t, f, x, y, time.Now())
		return shieldedBySmoke(covering, characterId, func() []uint32 {
			p, err := party.NewProcessor(l, ctx).GetByMemberId(characterId)
			if err != nil {
				// No party is the ordinary case for a solo player, not an
				// error: the caster's own mist has already been checked.
				l.WithError(err).Debugf("Smoke check: no party for character [%d].", characterId)
				return nil
			}
			ids := make([]uint32, 0, len(p.Members()))
			for _, m := range p.Members() {
				if !m.Online() {
					continue
				}
				ids = append(ids, m.Id())
			}
			return ids
		})
	}
}
```

In `character_damage.go`, add the dep:

```go
	inProtectiveMist   func(f field.Model, characterId uint32, x, y int16) bool
```

wire it in `CharacterDamageHandleFunc`'s `deps` literal:

```go
		inProtectiveMist:   newSmokeCheck(l, ctx, t),
```

and add the short-circuit in `processDamageTaken`, **immediately after** `characterId := c.Id()` and **before** the block sentinel:

```go
	// Smokescreen: a character standing in a protection mist owned by
	// themselves or an online party member takes nothing at all. This is a
	// SHORT-CIRCUIT, not another mitigation term, because that is what the
	// client does: CUserLocal::SetDamaged jumps to the function epilogue on a
	// positive IsSmokeAreaByPoint (v95 SetDamaged+0x1ef -> loc_93651F),
	// before the miss roll, Power Guard, Meso Guard, Achilles and Magic
	// Guard, and before the damage packet is built. Returning here is what
	// keeps reflect and Meso Guard amounts from being computed off damage the
	// shield zeroed (FR-4.5).
	//
	// Server-authoritative: the position comes from the character model, the
	// rectangle from the mist event, the party from the party service. An
	// honest client in smoke sends nothing, so this exists to stop a crafted
	// one claiming damage it did not take.
	if deps.inProtectiveMist != nil && deps.inProtectiveMist(f, characterId, c.X(), c.Y()) {
		l.Debugf("Character [%d] shielded by a protection mist in map [%d]; damage [%d] dropped.", characterId, f.MapId(), p.Damage())
		return
	}
```

The `!= nil` guard keeps every existing `character_damage_test.go` case working with its current `damageMitigationDeps` literals, which do not set the new field.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test -race ./socket/handler/ -v`
Expected: PASS — the new cases plus every pre-existing mitigation test unchanged.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler
git commit -m "feat(task-218): Smokescreen shields party members on the damage path"
```

---

### Task 16: Full verification sweep

**Files:** none created or modified unless a gate fails.

**Interfaces:** none — this is the CLAUDE.md gate list, run end to end before code review.

- [ ] **Step 1: Re-verify the seed templates (no change expected)**

FR: `AffectedAreaCreated` / `AffectedAreaRemoved` are already registered everywhere as of `ae3341511` (#1226, task-165) — re-verified, not assumed.

Run:
```bash
grep -rlc "AffectedAreaCreated" services/atlas-configurations/seed-data/templates/ | wc -l
grep -rlc "AffectedAreaRemoved" services/atlas-configurations/seed-data/templates/ | wc -l
ls services/atlas-configurations/seed-data/templates/ | wc -l
```
Expected: `11`, `11`, `11`. If any count differs, a template needs the writer registered before this task can ship — that is in scope, not a follow-up.

- [ ] **Step 2: Tests and vet, every changed module**

Run:
```bash
(cd libs/atlas-constants && go test -race ./... && go vet ./...)
(cd libs/atlas-constants/gen && go test -race ./... && go vet ./... && go run . -check)
(cd services/atlas-maps/atlas.com/maps && go test -race ./... && go vet ./... && go build ./...)
(cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...)
```
Expected: all clean; `go run . -check` prints `OK: … are up to date`.

- [ ] **Step 3: Guards**

Run:
```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/buff-duration-guard.sh
tools/skill-job-id-guard.sh
tools/trade-contract-mirror-guard.sh
tools/mist-contract-mirror-guard.sh
```
Expected: every one exits 0. `buff-duration-guard.sh` should be trivially clean — the recovery path deliberately emits `CHANGE_MP`, not a buff apply; if it fires, something was rewritten into `COMMAND_TOPIC_CHARACTER_BUFF` and that is a design regression, not a guard to silence.

- [ ] **Step 4: Lint**

Run: `tools/lint.sh --check`
Expected: clean. Needs nvm-provided node on PATH or it false-fails on the atlas-ui half; run `tools/lint.sh` (no flags) first to fix formatting in place.

- [ ] **Step 5: Container builds**

Run: `docker buildx bake atlas-maps atlas-channel`
Expected: both succeed. No `go.mod` gained a dependency in this task, but the shared root `Dockerfile` is the only thing that catches a missing `COPY libs/...`, and `go build` against the workspace will not.

- [ ] **Step 6: Record the residual open items**

Append to `docs/tasks/task-218-player-cast-mists/context.md` §8 the outcome of any in-game confirmation performed (the `x` absolute-vs-percentage question, and one live cast each of Smokescreen and Recovery Aura to confirm the USE_SKILL registration by looking for the handler's `Infof` line). If they were not performed, leave them listed as open — do not mark them settled.

- [ ] **Step 7: Code review, then PR**

Run `superpowers:requesting-code-review` (it dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer`; no atlas-ui TypeScript changed). Findings land in `docs/tasks/task-218-player-cast-mists/audit.md`. Address them, then open the PR.

- [ ] **Step 8: Commit any fixes**

```bash
git add -A
git commit -m "chore(task-218): verification sweep fixes"
```

---

## Requirement coverage

| Requirement | Task |
|---|---|
| FR-0.1 – FR-0.5 (Evan snapshot re-drain, additive-only, PROVENANCE) | 1, 2 |
| FR-1.1 (non-zero duration + non-degenerate rect per skill) | 8 (validation), 10–12 (per-skill real values in tests) |
| FR-1.2 (Recovery Aura magnitude from WZ `x`) | 12 |
| FR-1.3 (Flame Gear status derived from WZ = POISON) | 10 |
| FR-1.4 (data defects reported, never hard-coded around) | `design.md` §1.5; `context.md` §8 item 3 |
| FR-2.1, FR-2.2 (PROTECTION, RECOVERY kinds) | 3 |
| FR-2.3 (empty default stays DISEASE/CHARACTER) | 3 |
| FR-2.4 (explicit magnitude field, no DiseaseValue overload) | 3, 8, 12 |
| FR-2.5 (unknown kind rejected with a named warning) | 3 (create), 6 (tick, defence in depth) |
| FR-3.1 – FR-3.4 (`nType` 2, four-outcome derivation + test) | 4 |
| FR-4.1 – FR-4.6 (Smokescreen protection, scope, lag, composition) | 13, 14, 15 |
| FR-5.1 – FR-5.4 (party recovery, scope, caps, command-not-mutation) | 5, 6, 12 |
| FR-6.1 – FR-6.4 (monster DoT mists, POISON 0, window P−T) | 8, 10 |
| FR-7.1 – FR-7.6 (registry choice, Identity keying, wire id, blank imports, shared factoring) | 8, 9, 10, 11, 12 |
| FR-8.1 – FR-8.4 (reject don't clamp, no rollback, log-and-continue) | 8 |
| §5 mirror discipline + mirror guard (open question 8) | 7 |
| §8 NFRs (tick cost, bounded lifetime, concurrency, tenancy, observability, security) | 3 (defensive copy), 6, 8, 13, 15 |
| §10 acceptance: regression (Poison Mist, AREA_POISON, APPLY_STATUS keys) | 9 (untouched tests), 3, 4, 6 |
| §10 acceptance: build & verification gates | 16 |

