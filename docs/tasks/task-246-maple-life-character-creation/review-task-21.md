# Review: Task 21 — Mirror the `mapleLife` block into atlas-character-factory

Commit reviewed: `22a3e88` (single commit, `feat(atlas-character-factory): mirror the
Maple Life configuration block`).

## Scope

`git show --stat 22a3e88`:

```
 .../configuration/projection/projection_test.go    | 62 +++++++++++++++++
 .../configuration/tenant/maplelife/rest.go         | 78 ++++++++++++++++++++++
 .../character-factory/configuration/tenant/rest.go |  2 +
 3 files changed, 142 insertions(+)
```

Matches the brief's `### Files` list exactly (one new file, one field addition,
one test addition; `state.go`/`registry.go` untouched as claimed). No files
outside `services/atlas-character-factory` touched by this commit. `git status`
shows only pre-existing untracked task docs, unrelated to this commit.

## 1. The tag seam (priority 1)

Re-ran the implementer's normalized diff myself:

```
diff <(sed '1s/.*/package X/' services/atlas-configurations/.../tenants/maplelife/rest.go) \
     <(sed '1s/.*/package X/' services/atlas-character-factory/.../tenant/maplelife/rest.go)
```

Output: empty, exit 0. Confirmed independently — not just accepted from the
report. The two files are identical apart from the package clause, so every
struct, field name, type, and json tag (`gender`, `faces`, `hairs`,
`hairColors`, `skinColors`, `ordinal`, `jobId`, `level`, `mapId`, `stats`,
`ap`, `sp`, `spSkillId,omitempty`, `meso`, `equipment`, `inventory`, `looks`,
`classes`) matches Task 19's source (commit `f43e442`,
`services/atlas-configurations/atlas.com/configurations/tenants/maplelife/rest.go`)
byte-for-byte. **PASS.**

The `RestModel.MapleLife` field addition
(`services/atlas-character-factory/atlas.com/character-factory/configuration/tenant/rest.go:21`)
uses `json:"mapleLife"`, matching the corresponding field on the
`atlas-configurations` `tenants.RestModel` (Task 19). **PASS.**

## 2. The decode path actually reaches the new field (priority 2)

Read `configuration/projection/state.go` directly:

```go
func (s *State) ApplyTenant(env TenantEnvelope) error {
	var cfg tenant.RestModel
	if err := json.Unmarshal(env.Config, &cfg); err != nil {
```

This is a generic `json.Unmarshal` into the whole `tenant.RestModel` — no
explicit field list to update, so adding `MapleLife` to the struct is
sufficient. Confirmed by direct read, not assumed. **PASS.**

Read `configuration/registry.go` directly: `GetTenantConfig` (line 52) returns
`val, nil` where `val tenant.RestModel` comes straight from the
`tenantConfig` map populated by `PublishSnapshot`, which itself copies the
projection's `Snapshot()` map by value — the whole model, no projection. **PASS.**

`TestProjectionDecodesMapleLife`
(`configuration/projection/projection_test.go:121-182`, added by this commit)
calls `s.ApplyTenant(projection.TenantEnvelope{..., Config: cfgBts})` and then
`s.Snapshot()`, i.e. it drives the real `State.ApplyTenant` → `json.Unmarshal`
→ `Snapshot()` path, not an isolated `json.Unmarshal(&maplelife.RestModel{})`
call. It asserts on `snap[tid].MapleLife.Classes[0].JobId`,
`.SpSkillId`, and `.Looks[0].Faces` for the present case, and
`snap[tid].MapleLife.Classes` empty for the absent case. This is exactly the
decode path the brief asked the test to exercise. **PASS.**

## 3. Shape-only, no behaviour crept in (priority 3)

`configuration/tenant/maplelife/rest.go` contains only struct type
definitions (`LookOptions`, `StatBlock`, `EquipmentEntry`, `InventoryEntry`,
`ClassEntry`, `RestModel`) and doc comments — no methods, no validation, no
constructors. Verbatim match to Task 19's source, which was itself shape-only.
`tenant/rest.go`'s change is a single field + import addition, no logic.
**PASS.**

## Build / test verification

Ran independently (not just trusting the report's pasted output):

```
cd services/atlas-character-factory/atlas.com/character-factory && go build ./... && go test ./configuration/...
```

All packages `ok` or `[no test files]`, no failures. **PASS.**

## Findings

None blocking. None non-blocking.

## Not evaluable

None — the full review surface (tag parity, decode path, test honesty, shape
discipline) was directly verifiable within this commit and its two read-only
dependency files, all confirmed by independent commands rather than accepting
the implementer's report at face value.

## Verdict

APPROVED.
