# task-257 — Implementation Context

Companion to [plan.md](plan.md). PRD: [prd.md](prd.md). Design: [design.md](design.md).

## What this task actually changes

Three services, one behavior. `MOB_BANISH_PLAYER` currently decodes and logs and does nothing. After this task it emits a `BANISH` command that `atlas-monsters` validates against live field state and then executes through a shared `banishCharacter` helper, which the existing skill-129 path also adopts — so the WZ portal name and the WZ banish message land on both paths at once.

## Key files, by role

| Role | File | Note |
|---|---|---|
| Untrusted entry point | `services/atlas-channel/.../socket/handler/mob_banish_player.go:19` | today: `// behavior: deferred (decode-and-log only)`. The only stub this task removes |
| Wire contract (producer copy) | `services/atlas-channel/.../kafka/message/monster/kafka.go` | `BanishCommandBody` — hand-mirrored |
| Wire contract (consumer copy) | `services/atlas-monsters/.../kafka/consumer/monster/kafka.go` | `banishCommandBody` — hand-mirrored; edit both together |
| Trust boundary | `services/atlas-monsters/.../monster/processor.go` `Banish` | four ordered checks, fail-closed |
| Shared executor | `services/atlas-monsters/.../monster/processor.go` `banishCharacter` | the convergence point for both banish paths |
| Skill-129 path | `services/atlas-monsters/.../monster/processor.go:1247` `executeBanish` | rewritten onto the executor; target selection unchanged |
| Warp producer | `services/atlas-monsters/.../monster/disease.go:97-116` | `warpBody` + `warpCommandProvider`, widened by one argument; single existing caller |
| Warp consumer | `services/atlas-portals/.../portal/consumer.go:55` `handleWarpCommand` | gains precedence branch 3 |
| Name resolution | `services/atlas-portals/.../portal/processor.go` `WarpByName` | new; warn-and-fall-back, never drop |
| Banish data (read-only) | `services/atlas-monsters/.../monster/information/model.go:26` `Banish{Message, MapId, PortalName}` | already populated from `atlas-data` |

## Decisions carried from the design

- **Validation lives in `atlas-monsters`**, not `atlas-channel`. The channel's `data/monster` projection is template data; it cannot answer "is a mob of this template alive in this field" without a REST hop into `atlas-monsters` on the packet path, and it would fork banish into two implementations.
- **Envelope `monsterId: 0`, template id in the body.** `monsterId` means *unique* id everywhere else on `COMMAND_TOPIC_MONSTER`; `handleDamageCommand` and friends key off it.
- **`BANISH` keys on the character id**, unlike every other monster command in `atlas-channel`'s producer file (which key on the monster id). The ordering that matters is this character's banish requests against each other. This is a deliberate deviation, called out in the producer's doc comment and pinned by a test.
- **Warp first, then message** — design §7.3 resolves the PRD's internal §4.3 vs §9 contradiction in favour of §4.3's normative text. Flipping it later is a one-line change but a deliberate trade of a correctness guard for a cosmetic one.
- **`omitempty` on the producer side only.** `atlas-portals`' `warpBody` is never marshalled, so its field carries no `omitempty`; `atlas-monsters`' does, which is what keeps every existing `WARP` body byte-identical.
- **`monsterInformation` helper is added and used only at the new call sites.** The three existing `testInformationLookup` call sites (`processor.go:980`, `:1384`, `:1692`) are deliberately left alone — converting them is unrelated churn.
- **Out of scope by decision, not omission:** `ban/banType` (unverified semantics), the `potal` WZ typo in three nodes (falls through to `atlas-data`'s `"sp"` default), `atlas-channel`'s local `WarpBody` copy (no channel producer needs it), rate-limiting `MOB_BANISH_PLAYER`.

## Test seams

- **`atlas-monsters`:** `newRecordingProcessorWithBodies(t, ten)` (`monster/processor_test.go:236`) gives a `ProcessorImpl` whose `emit` records `{Topic, Type, Body}`; `testInformationLookup` (`processor.go:81`) stubs the information fetch; `GetMonsterRegistry().CreateMonster(...)` seeds live field state. `kill_test.go:22-71` is the closest complete template. Emitting through `p.emit` rather than `producer.ProviderImpl` inline is what makes `banishCharacter` testable from both callers.
- **`atlas-portals`:** `WarpToPortal` publishes to Kafka, which is absent under test — so the assertions are on which data-service paths the mock was asked for plus the logged warning, matching how the existing `Enter` tests work. `processor_test.go` is `package portal_test`; `consumer_test.go` is `package portal`, so the precedence test (which builds an unexported `warpEvent`) has to live in the latter.
- **`information.ModelBuilder` needs `SetBanish`** — it has no banish setter today, and steps 3–4 of the validation and both execution branches are unstubable without it.
- The `GetInField` lookup-error abort (check 1) has no dedicated test: the registry read is in-process and does not fail independently. The empty-registry case covers it structurally. Recorded here so its absence is a decision.

## Task sizing

Five tasks, each within the ~6-file / one-service guideline:

| Task | Service | Files |
|---|---|---|
| 1 | atlas-portals | 6 (incl. docs + 2 test files) |
| 2 | atlas-channel | 6 (incl. docs + 1 new test file) |
| 3 | atlas-monsters | 5 (plumbing: new package, producers, builder setter) |
| 4 | atlas-monsters | 5 (logic: validation, executor, consumer, docs) |
| 5 | — | 0 (verification gate only) |

Tasks 3 and 4 are both `atlas-monsters` and could have been one task. They are split because Task 3 is mechanical (a copied package, a widened signature, a builder setter) while Task 4 is the trust boundary and the shared-executor refactor. Splitting keeps the security-relevant review surface clean of copy-paste noise. Task 3 leaves the tree compiling and behaviorally correct on its own: it passes the portal name through the existing `executeBanish` call site, so the portal-name fidelity gap is already closed at the end of Task 3.

Task order matters: Task 1 (portals) before Task 3 (monsters producer) is not a compile dependency — the two sides are hand-copied structs, not a shared type — but landing the consumer first means no window where a `targetPortalName` is emitted that nothing reads.

## Verification

`tools/verify.sh` flagless must exit 0 before the branch is claimed done (Task 5). `--quick` / `--no-docker` also exit 0 but skip the bake and `-race`, so they do not count. Per-task verification is module-local `go build ./... && go test ./...` from the service's module root only.
