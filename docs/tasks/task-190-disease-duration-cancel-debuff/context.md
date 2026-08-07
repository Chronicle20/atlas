# task-190 — Implementation Context

Companion to [`plan.md`](plan.md). Inputs: [`prd.md`](prd.md), [`design.md`](design.md),
[`investigation.md`](investigation.md).

---

## 1. The two defects in one paragraph

`atlas-buffs` has interpreted the `COMMAND_TOPIC_CHARACTER_BUFF` `duration` field as
**milliseconds** since task-054 (`197324e40`, 2026-05-03), but two producers still send raw
WZ seconds — so every mob-applied disease expires ~1000× early. Because the buff is already
expired when `SET_TEMPORARY_STAT` is encoded, the client receives a stat born expired and
calls `CWvsContext::CheckTemporaryStatDuration`, sending serverbound `CANCEL_DEBUFF` at frame
rate to ask the server to drop it. That packet has no codec, no handler, and no template
routing anywhere in the repo, so nothing happens and the client wedges. Defect B is the more
consequential one: it makes **any** server/client temporary-stat disagreement unrecoverable.

Live evidence (`atlas-pr-1138`, GMS 83.1, 2026-08-04): mob 7130002 cast skill 126 emitting
`{"sourceId":126,"duration":15,...}`; 123 ms later the client began a `0x63` loop that ran
~1,500 packets over 3m44s, during which it stopped sending `CharacterBuffCancel` and attack
packets entirely.

---

## 2. Key files, verified

### FR-1 — duration units

| File | Site | Verified state |
|---|---|---|
| `services/atlas-data/atlas.com/data/mobskill/reader.go` | `readLevel`, `m.Duration = uint32(node.GetIntegerWithDefault("time", 0))` (≈`:66`) | no conversion — the fix goes here |
| `services/atlas-data/atlas.com/data/skill/reader.go` | `getEffect`, `e.SetDuration(e.Duration() * 1000)` (≈`:194-198`) | the precedent, with the "Why ms" comment to mirror |
| `services/atlas-monsters/.../monster/processor.go` | `buildMistCreateBody`: `durMs := int64(sd.Duration()) * int64(time.Second/time.Millisecond)` (≈`:1068`) | double-converts |
| same | `executeStatBuff`: `duration := time.Duration(sd.Duration()) * time.Second` (≈`:1105`) | double-converts |
| same | `executeDebuff`: `duration := int32(sd.Duration())` (≈`:1242`) | **correct after FR-1.1 — must not be edited** |
| `services/atlas-monsters/.../monster/processor.go` | `const MistDurationCapMs int64 = 60_000` (≈`:1050`) | today clamps a 1000×-inflated value, pinning **every** mob mist to exactly 60 s |
| `services/atlas-monsters/.../monster/disease.go` | `applyDiseaseCommandProvider` → `applyDiseaseBody{... Duration: duration ...}` (`:67-85`) | forwards a parameter; correct once callers are |
| `services/atlas-maps/.../tasks/mist_tick.go` | `Duration: int32(m.DiseaseDuration() / time.Second)` (`:86`) + the stale comment at `:81-86` | divides ms back to seconds |
| `services/atlas-maps/.../mist/processor.go:69` | `SetDisease(..., time.Duration(body.DiseaseDuration)*time.Millisecond)` | the mist model's `DiseaseDuration()` is a `time.Duration` built from ms — **this unit never changes**, so `mist/model_test.go:39` (`30*time.Second`) stays valid |
| `services/atlas-ui/src/components/features/monsters/MonsterSkillChip.tsx:114` | `` value: `${a.duration}s` `` | would render "15000s" after FR-1.1 |

Tests that pin the old seconds contract (locate by content, not line):
`atlas-monsters .../monster/processor_test.go` — `SetDuration(60)` ≈`:894`/`:1014`/`:1070`/`:1116`,
`SetDuration(10) // seconds` ≈`:1179`, `SetDuration(1800) // 30 minutes` ≈`:1236`,
`SetDuration(10)` ≈`:1263`; `atlas-maps .../tasks/mist_tick_test.go` ≈`:163`
(`require.Equal(t, int32(30), cmd.Body.Duration)` with a four-line seconds comment above it).

### FR-2 — CANCEL_DEBUFF

| File | What it gives you |
|---|---|
| `libs/atlas-packet/character/serverbound/chalkboard_close.go` | the empty-body serverbound codec precedent, method-for-method |
| `libs/atlas-packet/character/serverbound/chalkboard_close_test.go` | the `packet-audit:verify` marker style and per-version empty-body fixture style |
| `libs/atlas-packet/test/context.go:18` | `pt.Variants` — 12 entries incl. v48/61/72/79/83/84/86/87/92/95/JMS185 (and v28). **Append only**; existing code indexes `Variants[N]` positionally |
| `services/atlas-channel/.../socket/handler/mount_food_test.go` | the handler-test pattern actually used here: a decode pin + a `…HandleFunc` non-nil symbol test. There is no processor-injection seam in handlers |
| `services/atlas-channel/.../socket/handler/character_buff_cancel.go` | the handler shape: `func XHandleFunc(l, ctx, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{})` |
| `services/atlas-channel/.../shopscanner/registry.go` | the exact singleton registry shape to copy for `statreset` (`sync.Once` + `sync.RWMutex` + `Key{Tenant, CharacterId}` + `ClearCharacter`) |
| `services/atlas-channel/.../socket/init.go:49-55` | the destroyer where `shopscanner…ClearCharacter` already lives; `statreset` eviction goes beside it |
| `services/atlas-channel/.../main.go:880` | `handlerMap[charsb.CharacterBuffCancelHandle] = …` — registration goes right after |
| `services/atlas-channel/.../character/buff/{processor,producer}.go` | `Cancel` / `CancelByTypes` are the templates for `Expire` / `ExpireCommandProvider` |
| `services/atlas-buffs/.../character/registry.go:186` | `GetExpired(ctx, characterId) []buff.Model` — **already prunes and returns**; returns an empty slice on a Redis read failure |
| `services/atlas-buffs/.../character/processor.go:190` | `ExpireBuffs()` — the loop body to extract into `expireInto` |
| `services/atlas-buffs/.../kafka/consumer/character/consumer.go` | `InitHandlers` + the five `handleX` arms; `handleExpire` is a sixth of the same shape |
| `services/atlas-buffs/.../character/processor_test.go:18` | `setupProcessorTest(t) (Processor, tenant.Model, context.Context)` and `setupProcessorTestChanges()` |

### FR-3 — the guard

`tools/goroutineguard/` is the template, and it is a close one: `analyzer.go` (marker
collection + `markerFor` accepting the line or the line above), `cmd/goroutineguard/main.go`
(`singlechecker.Main`), `analyzer_test.go` (`analysistest.Run` over `testdata/src/{bad,good}`),
`go.mod` (`golang.org/x/tools v0.48.0`), and `tools/goroutine-guard.sh` (self-test → build once
→ sweep every `go.mod` under `services/` and `libs/` with `GOWORK=off`). CI job at
`.github/workflows/pr-validation.yml:138`, aggregated at `:713`/`:729`.

**`tools/` is deliberately not swept** — the analyzer's own testdata must be allowed to contain
the defective forms. `tools/goroutineguard` and `tools/rediskeyguard` are **absent from
`go.work`** (which lists only `catalog-lint`, `cideps`, `packet-audit`, `seed-splitters`);
`buffdurationguard` follows suit.

---

## 3. Decisions already made — do not relitigate

| # | Decision | Why |
|---|---|---|
| D1 | Reconcile via a **new `EXPIRE` Kafka command**, not sync REST | REST violates NFR-1 (cross-service HTTP on a ≤30/s hot path) and is racy — the channel would re-derive state atlas-buffs owns |
| D1 | Named `EXPIRE`, not `RECONCILE` | "reconcile" implies a two-way diff; the server does not diff, it prunes against server-side `expiresAt` |
| D2 | Throttle is **in-process, per (tenant, character), in atlas-channel** | a character's session lives on exactly one pod, so a per-pod map is the whole view. Redis would add a round-trip to the path whose purpose is to be cheap |
| D2 | Window = **1000 ms**, a `const` | caps a wedged client at 1 command/s (~30× reduction) while recovering 10× faster than the 10 s fleet sweep. Not env-configurable — deliberate non-goal |
| D2 | Throttle **before** the emit | the amplification NFR-2 bounds is the Kafka message; throttling in atlas-buffs is too late |
| D3 | Codec has **no version gates** | body is empty on all ten clients. The v48 3-arg `COutPacket(v5, 78, 0)` is a construction detail with no wire consequence |
| D3 | Placed in `character/serverbound` | matches its sibling `BuffCancelRequest`, even though the client function is on `CWvsContext` |
| D5 | Contract authority = **atlas-buffs `kafka/message/character/kafka.go`**, `ApplyCommandBody.Duration` | atlas-buffs is the consumer that defines the unit, so the unit is its property to declare. Producers get a one-line pointer, not a restatement |
| D6 | **Both** an analyzer and tests | an analyzer cannot see a *missing* multiplication (no signature); a test cannot cover a path nobody wrote yet. Neither alone meets "a reintroduced seconds emitter fails CI — demonstrated" |
| D6 | Analyzer fingerprints **json tag sets**, not type names | the body struct is duplicated under seven different local names |
| D6 | A typed unit (`type DurationMs int32`) was **rejected** | `DurationMs(15)` compiles as happily as `DurationMs(15000)`; it would churn seven services and enforce nothing |
| D7 | **No `docs/packets/registry/gms_v92.yaml`** | registry files are matrix-column artifacts (registry + IDA export + audit dir). v92 has a seed template but is not a column; a half-populated file would read as one |
| D8 | **No routing in `template_gms_12_1.json`** | it routes 24 handlers — login/select/map/move/chat/inventory/info/monster-move/NPC/summon — and no buff, skill, or attack handler at all |
| D9 | Four matrix cells are **`n-a`, not blank** | `status.json` asserts `"state": "n-a", "opcode": -1` for v48/61/72/79. Adding registry entries **corrects a wrong assertion** |

---

## 4. Non-obvious traps

1. **`executeDebuff` (`processor.go:1242`) must not be edited.** It is the one path that is
   *already* correct-by-forwarding once the reader lands. Adding a conversion there
   re-creates the bug in mirror image. Task 2 Step 5 and Task 15 Step 7 both check the diff.
2. **The 60 s mist cap changes meaning.** It currently clamps a 1000×-inflated value, so
   *every* mob mist is pinned to exactly 60 s. After the fix it clamps real milliseconds. The
   acceptance criterion "not clamped to exactly 60 s" is only testable against a mist skill
   whose authored `time ≤ 60` — execution must pick one and name it.
3. **WZ data is ingested, not parsed per request.** A code change in `mobskill/reader.go` does
   not fix already-ingested rows. Re-ingest, then **roll atlas-data** — `Storage.ByIdProvider`
   serves from a per-pod in-memory registry, so replicas that did not perform the ingest keep
   stale values. Skipping the roll is the most likely way for the fix to look like it failed.
   Verify the *data* (`GET /data/mob-skills/126` → `duration: 15000`), not the deploy.
4. **A template handler entry without a validator is silently dropped.** Indistinguishable
   from the packet never arriving. All ten entries need `"validator": "LoggedInValidator"`.
5. **`0x63` is not stable across versions.** It is `CANCEL_DEBUFF` at v83/v84 but the
   calc-damage-stat request at v61. Handlers bind by *name* through tenant config; a literal
   opcode anywhere in Go would mis-route.
6. **Five of ten clients never self-throttle.** v72/79/83/84/87 never assign
   `m_tLastStatResetRequest` in this function, so the 200 ms guard latches open and they send
   once per frame indefinitely. The client's floor is advisory; the server bound must be
   independent.
7. **`mist/model_test.go:39` is not affected.** It asserts `DiseaseDuration()` is
   `30*time.Second` — a `time.Duration` built from a ms input. That unit never changed; only
   the conversion *out* of it in `mist_tick.go` was wrong.
8. **The matrix `toolSha` reads git HEAD.** Regenerate the matrix *after* any rebase onto
   main, not before.
9. **`pt.Variants` is append-only.** Existing code indexes it positionally (`Variants[7]` =
   v48, etc.).
10. **The analyzer must follow one level of indirection.** The historical
    `processor.go:1068` defect lived in a local (`durMs := … * int64(time.Second/…)`), not
    inline in the composite literal — a value-expression-only check would miss it. That is
    also why the diagnostic lands on the assignment line, not the field line.
11. **`git stash` is shared across worktrees.** Never bare `git stash` / `git stash pop`; use
    a WIP commit, or `git stash push -u -m "<unique-tag>"` + `apply <sha>`.

---

## 5. Residuals accepted, and why

- **`AdaptHandler` opens an OTel span before the handler runs**, so a nudge flood still costs
  one span per packet even when throttled. That cost exists today for every unhandled packet
  and belongs to the shared dispatch layer, not this task.
- **Implementing FR-2 makes `0x6C` (`USER_CALC_DAMAGE_STAT_SET_REQUEST`) fire *more* often** —
  it is the tail of this handshake (`OnTemporaryStatReset` ends with
  `if (IsCalcDamageStat(mask)) { COutPacket(0x6C); Send… }`). It stays out of scope **on
  evidence**: one-shot per reset, not a loop, so it cannot wedge a client; the cost is a
  possibly-stale damage-range display. Filed as a follow-up at PR time (Task 14 Step 3), not
  dropped. Only three of ten opcodes are known (v48 `0x56`, v61 `0x63`, v83 `0x6C`) — note the
  v61 collision.
- **`tSwallowBuffTime` over-send** in `buff_cancel.go` — the client reads that trailing byte
  only when the mask contains a movement-affecting stat, and packets are length-framed, so
  the extra byte is simply left unread. An over-send worth tidying, **not** a desync. Recorded
  so a future reader does not re-raise it.
- **Cross-repo consumers of `mobskill.duration`** are unverified — it is an internal service
  API with no published contract. The design proceeds on the assumption that the in-repo set
  (atlas-monsters, atlas-maps, atlas-ui) is complete.
- **`MountBuffDuration = math.MaxInt32`** and any client tick-overflow it causes: separate
  concern, explicit PRD non-goal.

---

## 6. Verification quick reference

```bash
# per changed Go module
go build ./... && go vet ./... && go test -race ./...

# repo-root guards
./tools/redis-key-guard.sh
./tools/goroutine-guard.sh
./tools/buff-duration-guard.sh          # new, task-190
./tools/template-opcode-order-guard.sh  # templates changed
./tools/skill-job-id-guard.sh
./tools/lint.sh --check                 # fix mode: ./tools/lint.sh

# packet gates
cd tools/packet-audit && go run . matrix --check && go run . fname-doc --check

# UI (build type-checks tests; vitest alone is insufficient)
cd services/atlas-ui && source ~/.nvm/nvm.sh && nvm use 22 && npm run build && npx vitest run

# only if a SERVICE go.mod changed
docker buildx bake atlas-<svc>
```

`tools/service-registration-guard.sh` is not required — no service added, and `services.json`,
`deploy/k8s`, `docker-bake.hcl`, `go.work`, `tools/db-bootstrap.sh` are untouched. Confirm with
`git diff --name-only main...HEAD` before skipping it.

`tools/buffdurationguard/go.mod` is a new module but is **not** a service: it is absent from
`go.work` and `docker-bake.hcl`, needs no `COPY` line in the root `Dockerfile`, and needs no
bake.
