# task-299 — Mechanical Inventory

Captured at the branch point (`main` @ `31a791e3a`). If the branch drifts, regenerate
with the command shown above each section rather than trusting the listing.

---

## A. The 22 `tasks/task.go` copies (to delete)

```
$ md5sum $(find services -path '*tasks/task.go') | sort
```

Variant 1 — **divergent, no `ctx.Done()` check** (`48fe2ca06ba55eb1b190e7e6a24fc82a`, 4 files):

```
services/atlas-buffs/atlas.com/buffs/tasks/task.go
services/atlas-maps/atlas.com/maps/tasks/task.go
services/atlas-reactors/atlas.com/reactors/tasks/task.go
services/atlas-skills/atlas.com/skills/tasks/task.go
```

Variant 2 — **majority, cancellation-aware** (`e84da3270e946ddc3e4741a4ef79cadf`, 18 files):

```
services/atlas-account/atlas.com/account/tasks/task.go
services/atlas-ban/atlas.com/ban/tasks/task.go
services/atlas-channel/atlas.com/channel/tasks/task.go
services/atlas-character/atlas.com/character/tasks/task.go
services/atlas-doors/atlas.com/doors/tasks/task.go
services/atlas-drops/atlas.com/drops/tasks/task.go
services/atlas-expressions/atlas.com/expressions/tasks/task.go
services/atlas-guilds/atlas.com/guilds/tasks/task.go
services/atlas-invites/atlas.com/invites/tasks/task.go
services/atlas-login/atlas.com/login/tasks/task.go
services/atlas-merchant/atlas.com/merchant/tasks/task.go
services/atlas-monsters/atlas.com/monsters/tasks/task.go
services/atlas-mounts/atlas.com/mounts/tasks/task.go
services/atlas-pets/atlas.com/pets/tasks/task.go
services/atlas-rankings/atlas.com/rankings/tasks/task.go
services/atlas-rps/atlas.com/rps/tasks/task.go
services/atlas-summons/atlas.com/summons/tasks/task.go
services/atlas-world/atlas.com/world/tasks/task.go
```

The two bodies differ only in the loop:

```go
// variant 1 (4 services)                 // variant 2 (18 services)
for {                                     for {
    t.Run()                                   select {
    time.Sleep(t.SleepTime())                 case <-ctx.Done():
}                                                 l.Infof("Stopping task execution.")
                                                  return
                                              case <-time.After(t.SleepTime()):
                                                  t.Run()
                                              }
                                          }
```

Everything else — package clause, imports, `Task` interface, `Register` signature,
`routine.Go` wrapper — is identical.

---

## B. Task implementations — 44 files declaring both `Run()` and `SleepTime()`

```
$ grep -rl ') SleepTime() time.Duration' --include='*.go' services | xargs grep -l ') Run()'
```

```
services/atlas-account/atlas.com/account/account/task.go
services/atlas-ban/atlas.com/ban/ban/task.go
services/atlas-ban/atlas.com/ban/history/task.go
services/atlas-buffs/atlas.com/buffs/tasks/berserk.go
services/atlas-buffs/atlas.com/buffs/tasks/expiration.go
services/atlas-buffs/atlas.com/buffs/tasks/periodic.go
services/atlas-channel/atlas.com/channel/channel/task.go
services/atlas-channel/atlas.com/channel/character/combo/task.go
services/atlas-channel/atlas.com/channel/session/task.go
services/atlas-character/atlas.com/character/pending_change/task.go
services/atlas-character/atlas.com/character/session/task.go
services/atlas-doors/atlas.com/doors/door/expiry_task.go
services/atlas-drops/atlas.com/drops/drop/task.go
services/atlas-expressions/atlas.com/expressions/expression/task.go
services/atlas-guilds/atlas.com/guilds/guild/task.go
services/atlas-invites/atlas.com/invites/invite/task.go
services/atlas-login/atlas.com/login/session/task.go
services/atlas-maps/atlas.com/maps/tasks/jukebox.go
services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
services/atlas-maps/atlas.com/maps/tasks/respawn.go
services/atlas-maps/atlas.com/maps/tasks/weather.go
services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go
services/atlas-merchant/atlas.com/merchant/frederick/task.go
services/atlas-merchant/atlas.com/merchant/shop/task.go
services/atlas-monsters/atlas.com/monsters/character/hidden/task.go
services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go
services/atlas-monsters/atlas.com/monsters/monster/drop_timer_task.go
services/atlas-monsters/atlas.com/monsters/monster/picker_task.go
services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go
services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_task.go
services/atlas-monsters/atlas.com/monsters/monster/status_task.go
services/atlas-monsters/atlas.com/monsters/monster/task.go
services/atlas-mounts/atlas.com/mounts/mount/task.go
services/atlas-parcel/atlas.com/parcel/parcel/notification_task.go   <- OUT OF SCOPE
services/atlas-parcel/atlas.com/parcel/parcel/task.go                <- OUT OF SCOPE
services/atlas-pets/atlas.com/pets/pet/task.go
services/atlas-rankings/atlas.com/rankings/tasks/recompute.go
services/atlas-reactors/atlas.com/reactors/tasks/cooldown_cleanup.go
services/atlas-rps/atlas.com/rps/game/task.go
services/atlas-skills/atlas.com/skills/tasks/expiration.go
services/atlas-summons/atlas.com/summons/summon/beholder_task.go
services/atlas-summons/atlas.com/summons/summon/expiry_task.go
services/atlas-world/atlas.com/world/broadcast/task.go
services/atlas-world/atlas.com/world/channel/task.go
```

The two atlas-parcel files satisfy the interface shape but are **not** registered via
`tasks.Register`; they drive themselves with `routine.Go` plus their own ticker,
`stopCh`, and `WaitGroup` (see PRD §2 non-goals and OQ-3). `atlas-mts/task/periodic.go`
is the same third pattern.

Net in scope: **42 implementations across 22 services.**

---

## C. Implementations holding a captured `ctx` struct field (34 files)

```
$ xargs grep -ln 'ctx  *context.Context' < <impl-list>
```

These are the files where PRD FR-11 applies most directly — the body must switch from
the captured field to the `ctx` passed into `Run(ctx)`, and the field is removed if it
becomes unused.

```
services/atlas-ban/atlas.com/ban/ban/task.go
services/atlas-ban/atlas.com/ban/history/task.go
services/atlas-channel/atlas.com/channel/channel/task.go
services/atlas-channel/atlas.com/channel/character/combo/task.go
services/atlas-character/atlas.com/character/pending_change/task.go
services/atlas-character/atlas.com/character/session/task.go
services/atlas-doors/atlas.com/doors/door/expiry_task.go
services/atlas-drops/atlas.com/drops/drop/task.go
services/atlas-expressions/atlas.com/expressions/expression/task.go
services/atlas-guilds/atlas.com/guilds/guild/task.go
services/atlas-invites/atlas.com/invites/invite/task.go
services/atlas-maps/atlas.com/maps/tasks/jukebox.go
services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
services/atlas-maps/atlas.com/maps/tasks/respawn.go
services/atlas-maps/atlas.com/maps/tasks/weather.go
services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go
services/atlas-merchant/atlas.com/merchant/frederick/task.go
services/atlas-merchant/atlas.com/merchant/shop/task.go
services/atlas-monsters/atlas.com/monsters/character/hidden/task.go
services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go
services/atlas-monsters/atlas.com/monsters/monster/drop_timer_task.go
services/atlas-monsters/atlas.com/monsters/monster/picker_task.go
services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go
services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_task.go
services/atlas-monsters/atlas.com/monsters/monster/status_task.go
services/atlas-mounts/atlas.com/mounts/mount/task.go
services/atlas-parcel/atlas.com/parcel/parcel/notification_task.go   <- OUT OF SCOPE
services/atlas-parcel/atlas.com/parcel/parcel/task.go                <- OUT OF SCOPE
services/atlas-pets/atlas.com/pets/pet/task.go
services/atlas-rankings/atlas.com/rankings/tasks/recompute.go
services/atlas-summons/atlas.com/summons/summon/beholder_task.go
services/atlas-summons/atlas.com/summons/summon/expiry_task.go
services/atlas-world/atlas.com/world/broadcast/task.go
services/atlas-world/atlas.com/world/channel/task.go
```

---

## D. `tasks.Register` sites — 44 grep hits, 41 real call sites

```
$ grep -rn 'tasks\.Register' --include='*.go' services
```

Three hits are **doc comments, not calls** (update text per FR-15, do not rewrite as code):

```
services/atlas-character/atlas.com/character/pending_change/task.go:17
services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:294
services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:345
```

The 41 real call sites, by file:

| File | Calls | Context argument |
|---|---|---|
| atlas-account/main.go | 1 (:84) | `rt.Context()` |
| atlas-ban/main.go | 2 (:94, :97) | `rt.Context()` |
| atlas-buffs/main.go | 3 (:72, :75, :78) | `rt.Context()` |
| atlas-channel/main.go | 2 (:349, :363) | `rt.Context()` |
| atlas-character/main.go | 2 (:156, :160) | `rt.Context()` |
| atlas-doors/main.go | 1 (:97) | `ctx` |
| atlas-drops/main.go | 1 (:103) | `rt.Context()` |
| atlas-expressions/main.go | 1 (:49) | `rt.Context()` |
| atlas-guilds/main.go | 1 (:121) | `rt.Context()` |
| atlas-invites/main.go | 1 (:81) | `rt.Context()` |
| atlas-login/main.go | 1 (:171) | `rt.Context()` |
| atlas-maps/main.go | 4 (:133, :136, :139, :142) | `rt.Context()` |
| atlas-merchant/main.go | 3 (:107, :108, :109) | `rt.Context()` |
| atlas-monsters/main.go | 8 (:103–:110) | `ctx` |
| atlas-mounts/main.go | 1 (:113) | `rt.Context()` |
| atlas-pets/main.go | 1 (:117) | `rt.Context()` |
| atlas-rankings/main.go | 1 (:74) | `ctx` |
| atlas-reactors/main.go | 1 (:68) | `rt.Context()` |
| atlas-rps/main.go | 1 (:58) | `rt.Context()` |
| atlas-skills/main.go | 1 (:98) | `rt.Context()` |
| atlas-summons/main.go | 2 (:105, :106) | `ctx` |
| atlas-world/main.go | 2 (:142 `rt.Context()`, :150 `ctx`) | mixed |

Every one of the 22 services calls `service.Bootstrap` and already uses
`rt.WaitGroup()` elsewhere in `main.go` (consumer manager / server builder), so the
`wg` argument required by FR-2 is in scope at every call site with no plumbing.

The services passing a bare `ctx` (`atlas-doors`, `atlas-monsters`, `atlas-rankings`,
`atlas-summons`, `atlas-world`:150) bind it from `rt.Context()` earlier in `main`; no
behavior difference, only a local variable.

---

## E. Constructors that also take a `ctx` argument

Visible in section D: `NewExpiredBanCleanup`, `NewHistoryPurge`, `NewHeartbeat`,
`NewDecayTick`, `NewExpiryTask` (doors), `NewExpirationTask` (merchant shop),
`NewCleanupTask`, `NewNotificationTask`, `NewStatusExpirationTask`, `NewDropTimerTask`,
`NewSelfDestructTimerTask`, `NewMonsterAggroDecayTask`, `NewMonsterSkillPickerSweepTask`,
`NewMonsterRecoveryTask`, `NewReconciliationTask`, `NewRecomputeTask`,
`NewExpiryTask`/`NewBeholderTask` (summons), `NewExpiration` (world channel),
`NewSweep` (world broadcast).

Per FR-11 these signatures stay as they are unless the field becomes fully unused.

---

## F. Module wiring

All 22 services already declare in `go.mod`:

```
github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0-00010101000000-000000000000
replace github.com/Chronicle20/atlas/libs/atlas-routine => ../../../../libs/atlas-routine
```

`libs/atlas-routine/go.mod` requires only `logrus` (+ indirect `golang.org/x/sys`).
The scheduler adds `context`, `sync`, `time` — stdlib only. **No `go.mod`/`go.sum`
changes are expected.** `libs/atlas-routine` must not import `libs/atlas-service`
(cycle: `atlas-service/teardown.go` imports `atlas-routine`).
