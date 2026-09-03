# gms_12_1: `change_music`/`boat_effect` are a runtime no-op, not an excluded seed

`deploy/seed/gms/12_1/map-actions/onUserEnter/map-200090000.json` and
`map-200090010.json` exist and are byte-identical to the copies in every
other tenant seed directory. They are NOT excluded from `gms/12_1` — that was
a prior, abandoned approach (see "History" below). The dock-arrival map
actions they define still fire on `gms_12_1`; the `change_music` and
`boat_effect` operations they trigger are silently skipped and logged at
runtime because `gms_12_1` has no writer binding to encode them with.

## Why

Task C11 (`61307f074`) seeded these two documents into all 11 tenant
directories. Both use the `change_music` and `boat_effect` map-action
operations, which resolve through the tenant's template writer bindings for
`FieldEffect` and `ContiMove`.

Task C9b (`a4eff2415`) routed those two writers into every tenant that could
be derived from evidence, but `template_gms_12_1.json`
(`services/atlas-configurations/seed-data/templates/template_gms_12_1.json`)
still has no `FieldEffect` and no `ContiMove` writer entry. This repository
has no GMS v12 client binary, no IDA session, no
`docs/packets/ida-exports/gms_v12.json`, and no
`docs/packets/registry/gms_v12.yaml` from which to derive those opcodes. This
matches the ~20 prior packet-routing tasks that exclude `gms_12` for the
identical reason (see the `corpus_test.go` narrative in
`services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go`,
and the "gms_12 policy" note in
`libs/atlas-constants/gen/wzsnapshot/PROVENANCE.md`).

Sent unguarded, a writer miss is not a no-op: per
`libs/atlas-packet/resolve.go:55-72`, `ResolveCode` returns sentinel byte
`99` and logs that the resulting packet "will likely cause a client crash."

## History: the delete-the-seeds approach was blocked

Task C9c tried to resolve this by deleting the two seed documents from
`gms/12_1` only, leaving them in place for the other 10 tenants.
`tools/catalog-lint` (invoked by `tools/verify.sh`) enforces byte-identical
map-action seed replication across every `<region>/<version>` directory and
has no exemption mechanism, so removing the `gms_12_1` copies produced:

```
map-actions/onUserEnter/map-200090000.json: present in gms/48_1, missing from gms/12_1
map-actions/onUserEnter/map-200090010.json: present in gms/48_1, missing from gms/12_1
```

C9c was BLOCKED on this. The user, shown the catalog-lint constraint,
explicitly declined both a catalog-lint exemption mechanism and a
documentation-only gap (leaving the seeds in place with `gms_12_1` crashing
on encounter). The seed set therefore stays uniform across all 11 tenants,
and the version difference is absorbed at runtime instead.

## Resolution (task C9d)

`services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer.go`
gates `handleChangeMusic` and `handleBoatEffect` behind a per-tenant
writer-binding check before ever calling `session.Announce`:

- `changeMusicConfigured` looks up the tenant's `FieldEffect` writer options
  via `writer.TenantWriterOptions` and confirms the `BACKGROUND_MUSIC`
  operation key is present (`atlaspacket.CodeConfigured`).
- `boatEffectConfigured` does the same for the tenant's `ContiMove` writer
  and the `SHOW_STATE`/`HIDE_STATE` operation keys.

Either check failing (writer entirely unbound, as on `gms_12_1`, or writer
bound but missing the specific operation key) skips the write and logs a
`Warn`-level line naming the tenant/version, the operation, and the missing
writer — visible in normal operation, not a debug-only trace. No version
literal appears in this path: the guard reads whatever the tenant's template
actually bound at listener-build time, so a future tenant that also ships
without these writers is caught the same way, and a future `gms_12_1` fix
(deriving the opcodes and adding them to `template_gms_12_1.json`) makes the
guard pass with no code change here.

The ten tenants that DO bind `FieldEffect`/`ContiMove` are unaffected: the
guard passes and the write proceeds exactly as before task C9d.
`services/atlas-channel/atlas.com/channel/kafka/consumer/system_message/consumer_test.go`
pins both directions.

## If GMS v12 opcodes are ever derived

Add the `FieldEffect`/`ContiMove` writer entries (with their `operations`
tables) to `template_gms_12_1.json` once a GMS v12 client binary, IDA
session, `docs/packets/ida-exports/gms_v12.json`, or
`docs/packets/registry/gms_v12.yaml` becomes available. No change to the
seed set or to `consumer.go` is needed — the guard starts passing
automatically once the tenant's template binds the operations it checks for.
