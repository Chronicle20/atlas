# Touch-activated reactor templates

Authoritative list of `activateByTouch` reactor templates, as derived in Phase
2 of task-249 (`docs/tasks/task-249-touch-activated-reactors/design.md` §1.5).

## Source

This list is **not** a fresh re-grep performed for this task. It is the Phase
2 derivation recorded in design.md §1.5, enumerated from the Cosmic
`Reactor.wz` set (the WZ data Atlas reads for the v83-era Cosmic client) via:

```sh
grep -l 'activateByTouch' <WZ_ROOT>/Reactor.wz/*.img.xml
```

No `Reactor.wz` was mounted in this task's environment, so that command was
not re-run here; the enumeration below is quoted from design.md §1.5 rather
than re-confirmed independently.

## The ten templates

```
2406000  6109013  6109014  6109021  6109022  6109023
6109024  6109025  6109026  6109027
```

All ten carry `<int name="activateByTouch" value="1"/>` in the Cosmic
`Reactor.wz` set.

## Nine-item lists undercount

The nine-item lists that predate this task — the deferred bullet formerly at
`docs/TODO.md:280` and `docs/tasks/task-019-reactor-type-semantics/prd.md:32`
— omit `2406000` (나인스피릿의둥지, the Nine Spirit nest). That template is a
Horntail prequest reactor rather than a GPQ one, which is why the GPQ-focused
lists missed it. The ten-item list above, sourced from design.md §1.5, is the
authoritative correction.

## Per-mounted-WZ, not universal

The template count is per-mounted-WZ set, not a fixed MapleStory constant.
Other WZ sets present on the machine that produced design.md disagreed on
count: `atlas-ros` carried 10, `ms_1172` carried 9, and `AtlasMS` carried 19.
The ten templates above are authoritative for the Cosmic/v83-era data Atlas
reads, not universally across every client version or region.

## No type-100 event on any of the ten

None of the ten templates contains a type-100 event. `atlas-data` populates
`TL`/`BR` exclusively inside the `if t == 100` branch at
`services/atlas-data/atlas.com/data/reactor/reader.go:111`, reading the
event's `tl`/`rb` child vectors. With no type-100 event present, `TL` and
`BR` stay at their zero value — `(0,0)`/`(0,0)` — for every one of these ten
reactors.

This is why the touch-activation implementation (Task 6) does not reuse
`TL`/`BR` as a bounds source: it adds `touchAreaInfo`, derived from the
reactor's rendered layer rectangle, instead.
