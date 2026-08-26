# bug: @disease ZOMBIFY emits a non-existent temporary stat type

Task: task-256-zombify-healing-consequences (atlas-pr-1449)
Branch: task-256-zombify-healing-consequences
Filed: 2026-08-26

## Reproduced

Yes — live, environment `pr-1449` (channel + consumables from the PR overlay;
atlas-buffs and atlas-messages served from the `main` overlay, since
`atlas-pr-1449` deploys only channel, consumables, login and ingress).

Steps: as a GM, `@disease <name> zombify 1 <duration>` against self and another
character in the map.

## Observed

1. The command replies "Applied ZOMBIFY to target." but no zombify effect
   appears on either character.
2. An Elixir still restored the full 50% HP — not the halved amount FR-5/FR-6
   specify under zombify.

Live evidence, atlas-buffs (`atlas-main/atlas-buffs-d6b94cc4c-w2rf5`,
2026-08-26T14:53:49Z) — the APPLY command carries the stat type `ZOMBIFY`:

```
"body":{"fromId":0,"sourceId":0,"level":1,"duration":60,
        "changes":[{"type":"ZOMBIFY","amount":1}]}
```

Live evidence, atlas-channel (`atlas-pr-1449/atlas-channel-c497b758b-jh6ml`,
same timestamps) — the channel cannot resolve that name to a mask bit:

```
"message":"Attempting to add buff [ZOMBIFY], but cannot find it.",
"error":{"message":"character temporary stat type not found"}
```

## Expected

The command applies the ZOMBIFY disease as the `UNDEAD` temporary stat, the
client shows the effect, and `buff.IsZombified` (both services) sees it — so
HP-restoring consumables halve and Cleric Heal is negated.

## Root cause

`services/atlas-messages/atlas.com/messages/command/disease/commands.go:23-34`
declares `validDiseases` as an identity map from the user-typed disease word to
the stat-change `type` string put on the wire. For ZOMBIFY that identity is
wrong: `"ZOMBIFY"` is not a `character.TemporaryStatType`. The canonical name
for the zombify disease is `UNDEAD`
(`libs/atlas-constants/character/temporary_stat.go:122`), which is what every
other consumer of the value uses:

- `libs/atlas-packet/model/character_temporary_stat.go:264` —
  `CharacterTemporaryStatTypeByName` returns "character temporary stat type not
  found" for `ZOMBIFY`, so no mask bit is ever written to the client
  (symptom 1).
- `services/atlas-consumables/atlas.com/consumables/character/buff/model.go:73`
  and `services/atlas-channel/atlas.com/channel/character/buff/model.go:32` —
  both `IsZombified` predicates test `TemporaryStatTypeUndead`, so a buff
  carrying `ZOMBIFY` never matches (symptom 2).
- `services/atlas-buffs/atlas.com/buffs/character/immunity.go:10` — the
  disease/immunity set also keys on `UNDEAD`.
- `libs/atlas-constants/monster/skill.go:133` — the mob-inflicted path already
  emits `UNDEAD`, so mob zombify works and only the admin command is broken.

Repo-wide, the literal `"ZOMBIFY"` occurs in exactly one place: this table.

The same table has a second instance of the same defect: `"WEAKNESS"` is not a
`TemporaryStatType` either; the canonical name is `WEAKEN`
(`libs/atlas-constants/character/temporary_stat.go:36`). All nine other entries
(SEAL, DARKNESS, STUN, CURSE, POISON, SLOW, SEDUCE, CONFUSE, STOP_PORTION) do
match real constants.

Nothing in the task-256 diff is implicated — the branch's consumables and
channel logic is correct and simply never saw a zombify buff.

### Not a defect (test-procedure note)

`duration` is milliseconds (default `10000`). The observed command sent
`duration: 60`, i.e. a 60 ms buff. The regex at `commands.go:42` accepts only
digits for that argument, so a literal `60s` would not have matched at all.
Re-tests must pass `60000`.

## Fix

- `services/atlas-messages/atlas.com/messages/command/disease/commands.go` —
  make `validDiseases` an alias table mapping the user-typed word to the
  canonical `character.TemporaryStatType` value, sourced from
  `libs/atlas-constants/character` constants rather than bare string literals
  (per CLAUDE.md "check `libs/atlas-constants/` before defining a new … numeric
  constant"). `ZOMBIFY` → `TemporaryStatTypeUndead`; `WEAKNESS` →
  `TemporaryStatTypeWeaken`; the remaining nine map to their matching
  constants. Keep the typed word as the key so the existing GM syntax and the
  "Valid: …" help line are unchanged; accept the canonical names (`UNDEAD`,
  `WEAKEN`) as additional keys so both spellings work.
- `services/atlas-messages/atlas.com/messages/command/disease/commands_test.go`
  (new) — a table-driven test asserting that every value in `validDiseases`
  resolves via `packetmodel.CharacterTemporaryStatTypeByName` for a
  representative tenant (see
  `services/atlas-channel/atlas.com/channel/socket/model/character_test.go` for
  the tenant-construction idiom). This is the regression guard: it fails today
  for both ZOMBIFY and WEAKNESS. Add a case pinning that the parsed `ZOMBIFY`
  word produces an `UNDEAD` stat change on the emitted command.
- No change to atlas-consumables, atlas-channel, atlas-buffs or
  `libs/atlas-constants` — their side of the contract is already correct.

## Not yet answered

- Whether `libs/atlas-packet/model` is an acceptable test-only dependency for
  the atlas-messages module, or whether the test should instead assert against
  the `character.TemporaryStatType` constant set directly. Either satisfies the
  regression guard; prefer the constants-only form if it avoids adding a module
  dependency.

## Resolution

Fixed by `49465ebac` — `validDiseases` retyped to
`map[string]character.TemporaryStatType` and sourced from
`libs/atlas-constants/character`; `ZOMBIFY`→`TemporaryStatTypeUndead`,
`WEAKNESS`→`TemporaryStatTypeWeaken`, with `UNDEAD`/`WEAKEN` accepted as
additional command words. Regression test in `commands_test.go` asserts every
table value against its canonical constant and pins that the `ZOMBIFY` word
emits an `UNDEAD` stat change. `98424d31f` applied the gofmt import grouping the
lint guard required.

Review: APPROVED, 0 blocking / 0 non-blocking — see
`review-bug-disease-zombify.md`. The reviewer hand-traced the Kafka seam into
atlas-buffs (`character/immunity.go`), atlas-channel (mask encoding and
`character/buff.IsZombified`) and atlas-consumables
(`character/buff.IsZombified`), and confirmed no other producer emits the old
literals.

Gate: `tools/verify.sh --quick --base b2653c07c` — every check passes except the
lint & format guard, which aborts on a **pre-existing toolchain mismatch**
unrelated to this change: the pinned `GOLANGCI_LINT_VERSION=v2.12.2`
(`tools/lint.versions`) is built with go1.26 and panics with
`file requires newer Go version go1.27` against the local go1.27.0 toolchain.
Confirmed environment-wide, not branch-specific: `tools/lint.sh --check --go
services/atlas-buffs` — a module this branch never touches — panics identically.
Bumping the pin is a separate tooling change with tree-wide lint-finding
consequences; it is not made here.

**Live re-test: not yet performed.** The disease command lives in
atlas-messages, which the `atlas-pr-1449` overlay does not deploy — that
namespace serves atlas-messages from `main`. Re-testing this fix requires the
atlas-messages change to reach the namespace serving the command. Use a
millisecond duration, e.g. `@disease <name> zombify 1 60000`.
