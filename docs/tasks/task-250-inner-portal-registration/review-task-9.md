# Review: Task 9 — atlas-channel handler and handlerMap registration

Commit: `daa49c479` (range `bb38fa567..daa49c479`)
Brief: `.superpowers/sdd/plan/task-9-brief.md` (CONTROLLER AMENDMENTS applied)
Report: `.superpowers/sdd/plan/reports/task-9-report.md`

## Scope

Diff touches exactly the two files the brief names:

- `services/atlas-channel/atlas.com/channel/socket/handler/portal_inner.go` (new, 24 lines)
- `services/atlas-channel/atlas.com/channel/main.go` (+1 line, `handlerMap` registration)

`git show daa49c479 --stat` confirms no other files changed. Scope matches the
brief exactly — no scope mismatch.

## Directed checks

### 1. Handler shape matches `portal_script.go`

`socket/handler/portal_inner.go`:

```go
func InnerPortalHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := portal2.InnerPortal{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		_ = portal.NewProcessor(l, ctx).EnterInner(s.Field(), s.CharacterId(),
			p.PortalName(), p.X(), p.Y(), p.TargetX(), p.TargetY())
	}
}
```

vs `portal_script.go`'s `PortalScriptHandleFunc` — identical signature shape,
identical import set, identical decode → debug-log → delegate structure, `_ =`
discard convention on the delegated call. No business logic (no branching, no
distance math, no lookups) lives in the handler — that all lives in
`EnterInner`. PASS.

### 2. Argument order into `EnterInner`

Real accessors (`libs/atlas-packet/portal/serverbound/inner_portal.go:31-52`):
`PortalName()`, `X()`, `Y()`, `TargetX()`, `TargetY()`.

Real signature (`services/atlas-channel/atlas.com/channel/portal/processor.go:44-45`):

```go
EnterInner(f field.Model, characterId uint32, sourcePortalName string,
	claimedX int16, claimedY int16, claimedTargetX int16, claimedTargetY int16) error
```

Call site: `EnterInner(s.Field(), s.CharacterId(), p.PortalName(), p.X(), p.Y(), p.TargetX(), p.TargetY())`.

Positional mapping: `f=s.Field()`, `characterId=s.CharacterId()`,
`sourcePortalName=p.PortalName()`, `claimedX=p.X()`, `claimedY=p.Y()`,
`claimedTargetX=p.TargetX()`, `claimedTargetY=p.TargetY()`. Each field lands
in the position its name implies — no x/y↔targetX/targetY transposition.
PASS.

### 3. Tenant/character identity from ctx/session, not packet

`characterId` comes from `s.CharacterId()` (session), `f` (field, which
carries the tenant-scoped map context) comes from `s.Field()` (session).
`portal.NewProcessor(l, ctx)` resolves tenant from `ctx` internally
(confirmed in `processor.go:131`, `tenant.MustFromContext(p.ctx)`). The
decoded packet (`p`) contributes only `PortalName()`, `X()`, `Y()`,
`TargetX()`, `TargetY()` — all of which `EnterInner`'s own doc comment
(`processor.go:34-43`) states are used only for plausibility comparison and
logging, never adopted as authoritative. Matches PRD FR-3.4. PASS.

### 4. `handlerMap` registration — constant and collision check

`main.go:902`: `handlerMap[portal2.InnerPortalHandle] = handler.InnerPortalHandleFunc`,
placed immediately after `handlerMap[portal2.PortalScriptHandle] = handler.PortalScriptHandleFunc`
(line 901) as the brief directed.

`InnerPortalHandle` is declared in
`libs/atlas-packet/portal/serverbound/inner_portal.go:14` as
`const InnerPortalHandle = "InnerPortalHandle"` — this is the same package
(`portal2` alias) already imported for `PortalScriptHandle`, and it is the
constant the `InnerPortal` codec's own `Operation()` method returns
(`inner_portal.go:54-56`), so the handler is wired to the same handle the
codec declares.

Checked for collisions: `PortalScriptHandle = "PortalScriptHandle"` and
`InnerPortalHandle = "InnerPortalHandle"` are distinct string literals
(`libs/atlas-packet/portal/serverbound/script.go:13`,
`inner_portal.go:14`). `main.go` has 134 `handlerMap[...] = ...` assignments
into a single `map[string]handler.MessageHandler`; grep found no other
`= "InnerPortalHandle"` or `= "PortalScriptHandle"` definition elsewhere in
`libs/atlas-packet`. No shadowing. PASS.

Note: amendment #2 ("add the constant if missing") did not need to fire —
`InnerPortalHandle` already existed from the landed Task 3 codec commit. The
report calls this out explicitly and correctly; not a defect.

### 5. Discarded return value

`_ = portal.NewProcessor(l, ctx).EnterInner(...)` matches
`PortalScriptHandleFunc`'s `_ = portal.NewProcessor(l, ctx).Enter(...)`
convention exactly. `EnterInner`'s doc comment confirms every refusal path
returns `nil` and is a deliberate no-op (the only non-nil return would come
from the position-publish call inside `EnterInner`, which the handler is not
expected to react to any differently than `Enter` reacts to its own
error today — actually `Enter` does surface an error via `producer.ProviderImpl`
but the handler still discards it, so the convention for "discard whatever
`Processor.X` returns" is pre-existing and unchanged here). Not a defect per
the directed-check instruction.

## Build/test verification

Ran from module root `services/atlas-channel/atlas.com/channel`:

```
go build ./...   → clean
go vet ./socket/handler/...   → clean
```

(Full `go test ./...` output not independently re-run in this review beyond
build/vet, since the report already documents a clean 2162-test pass and this
commit adds no new test file — consistent with `git show --stat` showing only
the two non-test files above.)

## Findings

None. All five directed checks PASS with cited evidence.

## Not evaluable

- None. The full diff surface (2 files, 25 lines) was read in full; both
  referenced contracts (`InnerPortal` codec, `EnterInner` signature and doc
  comment) were read in full from source, not inferred.

## Verdict

APPROVED
