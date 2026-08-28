# Code-Derived Kafka Topic Manifest — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the set of Kafka topics a code-derived artifact — a `topic.Token`-typed constant in Go is the single source of truth, a generator renders every deployment surface from it, an analyzer gates drift, and `atlas-kafka-precreate` reads the generated manifest instead of prefix-scraping `os.Environ()`.

**Architecture:** A new `topic.Token` defined type marks every topic-token constant. A new module `libs/atlas-kafka/gen` loads the whole `go.work` workspace through `golang.org/x/tools/go/packages`, collects every `topic.Token` constant by type, and writes `topics.yaml` plus four rendered deployment artifacts by marker splice. A sixth go/analysis analyzer (`tools/topicguard`) joins the existing shared vet sweep and rejects bare literals, raw `os.Getenv` topic reads, and tokens missing from the manifest — that analyzer, not the expensive generator, is the gate on the common `.go` change path. The ~346-file retype and the 336-site `EnvProvider` error sweep are performed by a purpose-built AST codemod (`tools/topicmod`), not by hand.

**Tech Stack:** Go 1.27.0, `golang.org/x/tools v0.49.0` (`go/analysis`, `go/packages`, `go/ast/astutil`), `gopkg.in/yaml.v3 v3.0.1`, bash + bats for wrapper scripts, Kustomize overlays, ArgoCD sync waves.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md))

## Global Constraints

- Go toolchain is **`go 1.27.0`**; every new `go.mod` declares `go 1.27.0`.
- `golang.org/x/tools` is pinned at **`v0.49.0`** across `tools/*` modules — any new tool module MUST use exactly that version.
- YAML is `gopkg.in/yaml.v3 v3.0.1` (the version `libs/atlas-constants/gen/go.mod` already pins).
- Tool modules under `tools/` and the new `libs/atlas-kafka/gen` are **NOT** added to `go.work`; they build and test with `GOWORK=off`. (`libs/atlas-constants/gen` IS in `go.work` at `go.work:5` — do not copy that; copy `tools/atlasguards`' posture instead.)
- **Never normalize line endings.** All rewrites are marker splices that preserve every byte outside the marked region.
- Repo-relative paths only in committed files. No absolute or home paths.
- **No placeholders**: no `// TODO`, no stubbed handler, no unimplemented status response. A site the codemod cannot rewrite goes into an explicit residue list, never a silent skip.
- The topic-token shape, used identically by the generator, the analyzer, and the codemod, is the regexp `^[A-Z0-9_]*TOPIC[A-Z0-9_]*$` applied to the constant's **value**.
- `topic.Token` is the NAME OF THE ENVIRONMENT VARIABLE, never the resolved topic name. Functions taking a resolved name (`consumer.NewConfig`'s `topic string`, `producer.WriterFactory`, `Manager.writers` map keys, `outbox.Message.Topic`, every `rf(t, handler)` registration) stay `string`. This distinction is load-bearing for the analyzer.

## Measured baseline (from design.md §1 and the code survey — do not re-derive)

| Fact | Value | Source |
|---|---|---|
| Non-test files declaring a topic token | **346** (all under `services/`; **zero** under `libs/`) | survey §F |
| Files with the most declarations | `services/atlas-channel` 70, `services/atlas-saga-orchestrator` 36, `services/atlas-maps` 12 | survey §F |
| Declaration lines | 517 non-test lines, 159 distinct tokens | design §1 |
| `topic.EnvProvider` call sites | **336** (334 in `services/`, 2 in `libs/atlas-kafka`, 1 in `libs/atlas-outbox`) | survey §G |
| Per-service message `Buffer` types | **42** (41 services + `libs/atlas-service`) | survey §H |
| True error-sweep residue (enclosing func returns neither `error` nor is a `NewConfig` wrapper) | **exactly 1**: `services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go:52` | survey §G |
| Bare string literal reaching a token parameter | **exactly 1**: `services/atlas-marriages/atlas.com/marriages/kafka/consumer/character/consumer.go:78` | survey §G, design §4 |
| Raw `os.Getenv` topic reads | **exactly 4**: `libs/atlas-service/envregistry.go:52,59`, `libs/atlas-service/projection.go:67,68` | survey §J |
| Topic keys in `deploy/k8s/base/env-configmap.yaml` | 174, spanning lines 21–196, split by `DB_HOST`/`DB_PORT` at **103–104** | survey §I, verified |
| Topic literals per overlay | `main` 60–233, `pr` 180–353, `pr-sparse` 343–516 — 174 each | survey §I |
| Topic lines in `deploy/compose/.env.example` | 89, lines 20–112 | survey §I |
| Expected token count after regeneration | 159 (174 − 17 orphans + 2 `STATUS_*`) | design §12 |

---

## Task 1: `topic.Token` type and the `EnvProvider` error contract

Introduces the type and removes the silent fallback. **No signature changes** — `EnvProvider` keeps `token string` for now, so this task compiles and tests green across the whole workspace. The retype happens in Task 4 via the codemod.

Behaviour change landing here: `EnvProvider` now returns an error instead of the token when the variable is unset or empty. The 325 call sites that discard that error still compile; they are fixed in Task 4. This is the one intentionally red-at-runtime interval on the branch — noted in `context.md`.

### Files

- `libs/atlas-kafka/topic/token.go` — **new file**; the `Token` defined type
- `libs/atlas-kafka/topic/provider.go` — remove the fallback at lines 17–21, return an error
- `libs/atlas-kafka/topic/provider_test.go` — **new file**; `libs/atlas-kafka/topic/` has no test file today

Module root for `go build`/`go test`: `libs/atlas-kafka`.

Patterns to copy: `libs/atlas-kafka/consumergroup/resolver_test.go` (same module, table-driven, no external fixtures).

- [ ] **Step 1: Write the failing test**

`TestEnvProvider` in `libs/atlas-kafka/topic/provider_test.go`, package `topic` (internal — no fixtures needed). Use `t.Setenv` for isolation. Logger: `logrus.New()` with `SetOutput(io.Discard)`.

| subtest name | env setup | expect value | expect error |
|---|---|---|---|
| `resolved` | `EVENT_TOPIC_PROVIDER_TEST=evt-resolved` | `"evt-resolved"` | nil |
| `unset` | variable not set | `""` | non-nil, `err.Error()` contains `EVENT_TOPIC_PROVIDER_TEST` |
| `empty value` | `EVENT_TOPIC_PROVIDER_TEST=` | `""` | non-nil, `err.Error()` contains `EVENT_TOPIC_PROVIDER_TEST` |

The exact error text asserted by the `unset` and `empty value` cases:

```
topic token [EVENT_TOPIC_PROVIDER_TEST] has no value in the environment
```

Assert with `err.Error() == want` for both error cases (identical message for unset and empty — the caller cannot act differently on the two).

```go
func TestEnvProvider(t *testing.T) {
	const tok = "EVENT_TOPIC_PROVIDER_TEST"
	const wantErr = "topic token [EVENT_TOPIC_PROVIDER_TEST] has no value in the environment"
	// table per the case table above; each case calls
	//   got, err := EnvProvider(l)(tok)()
	// and asserts got and err against the row.
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-kafka && go test ./topic/... -run TestEnvProvider -v`
Expected: FAIL — `unset` returns `("EVENT_TOPIC_PROVIDER_TEST", nil)` instead of `("", error)`.

- [ ] **Step 3: Write `token.go`**

```go
// libs/atlas-kafka/topic/token.go
package topic

// Token is the NAME OF THE ENVIRONMENT VARIABLE that carries a topic's
// per-environment name -- never the topic name itself. The distinction is
// load-bearing: overlays suffix the name, the manifest carries only the
// token, and every function that takes a resolved name keeps taking a
// plain string.
//
// Declare tokens as `X topic.Token = "COMMAND_TOPIC_Y"`. tools/topicguard
// rejects a bare literal reaching a Token parameter, and libs/atlas-kafka/gen
// collects every Token constant by type into libs/atlas-kafka/gen/topics.yaml.
type Token string
```

- [ ] **Step 4: Remove the fallback in `provider.go`**

Replace the body of the innermost closure (`provider.go:17-22`):

```go
		return func() (string, error) {
			t, ok := os.LookupEnv(token)
			if !ok || t == "" {
				return "", fmt.Errorf("topic token [%s] has no value in the environment", token)
			}
			return t, nil
		}
```

Add `"fmt"` to the import block; drop the now-unused `logrus` warn call but keep the `l logrus.FieldLogger` parameter (336 call sites pass it; removing it is out of scope).

- [ ] **Step 5: Run tests**

Run: `cd libs/atlas-kafka && go build ./... && go test ./... 2>&1 | tail -30`
Expected: PASS. Note: `libs/atlas-kafka/producer/manager_test.go` and `provider_test.go` may assert the old fallback — if so, update those assertions to the new error contract in this task (they are in the same module and the new behaviour is the spec).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-kafka/topic/token.go libs/atlas-kafka/topic/provider.go libs/atlas-kafka/topic/provider_test.go libs/atlas-kafka/producer
git commit -m "feat(atlas-kafka): add topic.Token and make EnvProvider error on an unset token"
```

---

## Task 2: `tools/topicmod` codemod — type rules (R1, R2)

The declaration retype spans 346 files across 64 service modules and the `Buffer` retype spans 42 more. Per `docs/codemod-vs-agents.md`, that clears the second-dispatch threshold decisively — this is a rewriter, not a fan-out. This task builds the module and the two **type** rules; Task 3 adds the two **error** rules; Task 4 runs it.

### Files

- `tools/topicmod/go.mod` — **new file**; module `github.com/Chronicle20/atlas/tools/topicmod`, `go 1.27.0`, requires `golang.org/x/tools v0.49.0`
- `tools/topicmod/rewrite.go` — **new file**; R1 + R2 rewrite logic over `*ast.File`
- `tools/topicmod/rewrite_test.go` — **new file**; table-driven over `testdata/`
- `tools/topicmod/testdata/r1_decl/{before,after}.go.txt` — **new files**
- `tools/topicmod/testdata/r2_buffer/{before,after}.go.txt` — **new files**
- `tools/topicmod/cmd/topicmod/main.go` — **new file**; CLI: `topicmod [-check] <dir>...`
- `libs/atlas-kafka/topic/token.go` — **new file** in Task 1; read-only here — the type this codemod introduces at call sites
- `services/atlas-marriages/atlas.com/marriages/kafka/message/character/kafka.go` — read-only; the canonical R1 input shape (lines 9-13)
- `services/atlas-buffs/atlas.com/buffs/kafka/message/buffer.go` — read-only; the canonical R2 input shape (lines 14-59)

Module root: `tools/topicmod` (build/test with `GOWORK=off`).

Patterns to copy: `tools/scopeguard/go.mod` (x/tools pin and module-path shape). The rewriter is `astutil`-based, not `analysis`-based — it writes files; only `tools/topicguard` (Task 9) is an analyzer.

### Interfaces

- Produces, consumed by Task 3 and Task 4:
  - `func Rewrite(fset *token.FileSet, f *ast.File, path string) (changed bool, residue []Finding)`
  - `type Finding struct { Pos token.Position; Rule string; Reason string }`
  - `func Run(dirs []string, check bool) ([]Finding, error)` — walks `.go` files (skipping `_test.go`, `vendor/`, `testdata/`), applies `Rewrite`, formats with `go/format`, and writes unless `check`.

- [ ] **Step 1: Write the failing test**

`TestRewrite` in `tools/topicmod/rewrite_test.go`, table-driven over `testdata/<case>/before.go.txt` → `after.go.txt`. Each case parses `before.go.txt`, calls `Rewrite`, formats the result with `format.Node`, and compares byte-for-byte against `after.go.txt`. Cases: `r1_decl`, `r2_buffer`.

`testdata/r1_decl/before.go.txt`:

```go
package character

const (
	EnvEventTopicStatus    = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated = "CREATED"
	StatusEventTypeDeleted = "DELETED"
)
```

`testdata/r1_decl/after.go.txt`:

```go
package character

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicStatus    topic.Token = "EVENT_TOPIC_CHARACTER_STATUS"
	StatusEventTypeCreated             = "CREATED"
	StatusEventTypeDeleted             = "DELETED"
)
```

Three properties this case pins: only the TOPIC-shaped value is retyped; `StatusEventTypeCreated`/`Deleted` are untouched; the `topic` import is added exactly once even for multiple retyped names in one block.

`testdata/r2_buffer/before.go.txt`:

```go
package message

import (
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Buffer struct {
	buffer map[string][]message.Message
}

func (b *Buffer) Put(t string, p model.Provider[[]message.Message]) error {
	ms, err := p()
	if err != nil {
		return err
	}
	b.buffer[t] = append(b.buffer[t], ms...)
	return nil
}

func (b *Buffer) GetAll() map[string][]message.Message {
	return b.buffer
}
```

`testdata/r2_buffer/after.go.txt` — identical except `map[string][]message.Message` becomes `map[topic.Token][]message.Message` in both the field and the `GetAll` result, `Put`'s first parameter becomes `t topic.Token`, and the `topic` import is added.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/topicmod && GOWORK=off go test ./... -run TestRewrite -v`
Expected: FAIL — `Rewrite` undefined.

- [ ] **Step 3: Implement R1 and R2**

**R1 — declaration retype.** For each `*ast.GenDecl` with `Tok == token.CONST`, for each `*ast.ValueSpec` with `spec.Type == nil` and exactly one value that is a `*ast.BasicLit` of kind `token.STRING`: if the unquoted value matches `^[A-Z0-9_]*TOPIC[A-Z0-9_]*$`, set `spec.Type` to `&ast.SelectorExpr{X: ast.NewIdent("topic"), Sel: ast.NewIdent("Token")}`. A `ValueSpec` carrying several names on one line where only some values are topic-shaped is **residue**, reason `"mixed const spec"` — do not attempt to split it.

**R2 — Buffer retype.** Within a file that declares `type Buffer struct` with a field of type `map[string][]<kafka message>`: retype that field's key, the result type of `func (b *Buffer) GetAll()`, and the first parameter of `func (b *Buffer) Put(t string, …)` to `topic.Token`. Also retype the loop variable use in a top-level `Emit`/`EmitWithResult` that ranges over `b.GetAll()` — no edit is needed there because the range variable's type is inferred; assert only that no explicit `var t string` shadows it, and report residue reason `"explicit string binding over GetAll range"` if one does.

Import insertion for both rules: `astutil.AddImport(fset, f, "github.com/Chronicle20/atlas/libs/atlas-kafka/topic")`, idempotent.

`Run` skips `_test.go` (FR-1.7 is structural), `vendor/`, and `testdata/`.

- [ ] **Step 4: Run tests**

Run: `cd tools/topicmod && GOWORK=off go test ./... -v`
Expected: PASS, both cases.

- [ ] **Step 5: Commit**

```bash
git add tools/topicmod
git commit -m "feat(topicmod): AST codemod for topic.Token declaration and buffer retyping"
```

---

## Task 3: `tools/topicmod` — error rules (R3, R4)

Adds the `EnvProvider` error sweep to the same rewriter. 336 call sites in two shapes (design §11).

### Files

- `tools/topicmod/rewrite.go` — **new file** in Task 2; add R3 and R4 here
- `tools/topicmod/rewrite_test.go` — **new file** in Task 2; add the two cases here
- `tools/topicmod/testdata/r3_propagate/{before,after}.go.txt` — **new files**
- `tools/topicmod/testdata/r4_newconfig/{before,after}.go.txt` — **new files**
- `services/atlas-marriages/atlas.com/marriages/kafka/consumer/character/consumer.go` — read-only; the canonical R3 (lines 30-43) and R4 (lines 25-28) input shapes

Module root: `tools/topicmod`.

### Interfaces

- Consumes from Task 2: `Rewrite`, `Finding`, `Run` — unchanged signatures.

- [ ] **Step 1: Write the failing test**

Two more rows in `TestRewrite`'s table.

`testdata/r3_propagate/before.go.txt`:

```go
package character

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, h handler.Handler) (string, error)) error {
	return func(rf func(topic string, h handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(characterMsg.EnvEventTopicStatus)()
		if _, err := rf(t, h1); err != nil {
			return err
		}
		t, _ = topic.EnvProvider(l)(characterMsg.EnvEventTopicOther)()
		if _, err := rf(t, h2); err != nil {
			return err
		}
		return nil
	}
}
```

`testdata/r3_propagate/after.go.txt`:

```go
package character

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, h handler.Handler) (string, error)) error {
	return func(rf func(topic string, h handler.Handler) (string, error)) error {
		var t string
		var err error
		t, err = topic.EnvProvider(l)(characterMsg.EnvEventTopicStatus)()
		if err != nil {
			return err
		}
		if _, err := rf(t, h1); err != nil {
			return err
		}
		t, err = topic.EnvProvider(l)(characterMsg.EnvEventTopicOther)()
		if err != nil {
			return err
		}
		if _, err := rf(t, h2); err != nil {
			return err
		}
		return nil
	}
}
```

This case pins the multi-handler variant specifically: one hoisted `var err error`, `t, _ =` becomes `t, err =` (assignment, not `:=`, because `t` is already declared), and each site gains its own guard.

A second sub-case in the same file family, `r3_propagate` case 2 — the short-declaration variant `t, _ := topic.EnvProvider(l)(X)()` inside a func returning `error` becomes `t, err := topic.EnvProvider(l)(X)()` followed by `if err != nil { return err }`, with **no** hoisted `var err error`.

`testdata/r4_newconfig/before.go.txt`:

```go
package consumer

func NewConfig(l logrus.FieldLogger) func(name string) func(token string) func(groupId string) consumer.Config {
	return func(name string) func(token string) func(groupId string) consumer.Config {
		return func(token string) func(groupId string) consumer.Config {
			t, _ := topic.EnvProvider(l)(token)()
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(brokers(), name, t, groupId)
			}
		}
	}
}
```

`testdata/r4_newconfig/after.go.txt`:

```go
package consumer

func NewConfig(l logrus.FieldLogger) func(name string) func(token topic.Token) func(groupId string) consumer.Config {
	return func(name string) func(token topic.Token) func(groupId string) consumer.Config {
		return func(token topic.Token) func(groupId string) consumer.Config {
			t, err := topic.EnvProvider(l)(token)()
			if err != nil {
				l.WithError(err).Fatalf("unresolvable topic token [%s]", token)
			}
			return func(groupId string) consumer.Config {
				return consumer.NewConfig(brokers(), name, t, groupId)
			}
		}
	}
}
```

Note what is NOT retyped: `consumer.NewConfig(brokers(), name, t, groupId)`'s third argument `t` is the **resolved name** and stays `string`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/topicmod && GOWORK=off go test ./... -run TestRewrite -v`
Expected: FAIL on the two new cases.

- [ ] **Step 3: Implement R3 and R4**

**R4 first (it is the narrower match).** A `*ast.FuncDecl` named `NewConfig` whose result type is the curried `func(name string) func(token string) func(groupId string) consumer.Config` chain: retype every `token string` parameter in the chain (declaration and both nested `*ast.FuncType`s) to `token topic.Token`, and replace the enclosed `t, _ := topic.EnvProvider(l)(token)()` with the assignment plus the `Fatalf` guard shown above.

**R3.** For every assignment (`:=` or `=`) whose RHS is a call of the shape `topic.EnvProvider(<any>)(<any>)()` and whose LHS second element is the blank identifier `_`:
- Find the innermost enclosing `*ast.FuncDecl` or `*ast.FuncLit`. If its result list is empty or its last result is not `error` → **residue**, reason `"enclosing function does not return error"`. Do not rewrite.
- If the assignment is `:=` → rewrite `_` to `err` and insert `if err != nil { return err }` immediately after.
- If the assignment is `=` → rewrite `_` to `err`, insert the same guard, and ensure a `var err error` declaration exists in the same block; add one immediately after the existing `var t string` if absent.
- Skip anything already handled by R4.

Rule ordering in `Rewrite`: R4, R3, R1, R2. Residue accumulates across rules.

- [ ] **Step 4: Run tests**

Run: `cd tools/topicmod && GOWORK=off go test ./... -v`
Expected: PASS, four cases.

- [ ] **Step 5: Commit**

```bash
git add tools/topicmod
git commit -m "feat(topicmod): rewrite discarded EnvProvider errors and NewConfig token wrappers"
```

---

## Task 4: Run the codemod across the repository

The one deliberately large task (see `context.md`). It is tool-driven, not per-file: two `topicmod` invocations, four hand edits, and a build sweep. Lands as **two commits** so the mechanical retype and the error handling are separately reviewable (design §14).

### Files

- `services/` — 346 declaration files + 42 buffer files + 61 `NewConfig` wrappers, all rewritten by the `tools/topicmod` binary (**new file** in Task 2); do not hand-edit
- `libs/atlas-kafka/producer/provider.go` — hand edit: `type Provider func(token topic.Token) MessageProducer` (line 10)
- `libs/atlas-kafka/producer/manager.go` — hand edit: `func (m *Manager) Writer(l logrus.FieldLogger, token topic.Token) (Writer, error)` (line 66) and `ManagerWriterProvider(l) func(token topic.Token) model.Provider[Writer]` (line 133); `m.writers` map keys stay `string` (resolved names)
- `libs/atlas-outbox/bridge.go` — hand edit: `contents map[topic.Token][]kafka.Message` (line 21); `Message.Topic` at line 42 stays `string`
- `services/atlas-marriages/atlas.com/marriages/kafka/consumer/character/consumer.go` — hand edit line 78: replace the bare literal `"EVENT_TOPIC_MARRIAGE_STATUS"` with `marriageMsg.EnvEventTopicStatus`, adding the import `marriageMsg "atlas-marriages/kafka/message/marriage"` (the constant is declared at `services/atlas-marriages/atlas.com/marriages/kafka/message/marriage/kafka.go:13`)
- `services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go` — hand edit: the sole residue site at line 52; `(*NotificationTask).Run()` returns nothing, so keep the existing log-and-return, but change the discarded binding to a named `err` so the analyzer sees a handled error
- `libs/atlas-service/envregistry.go` — hand edit lines 52, 59
- `libs/atlas-service/projection.go` — hand edit lines 67, 68
- `libs/atlas-service/topics.go` — **new file**; the four `topic.Token` constants those two files read

Module roots: every module in `go.work` (95 entries). Build sweep uses `tools/test-all-go.sh`.

Patterns to copy: `libs/atlas-service/go.mod:7,57` already requires and replaces `libs/atlas-kafka`, so no go.mod change is needed for the new import.

### Interfaces

- Consumes from Tasks 2–3: `tools/topicmod/cmd/topicmod` CLI.
- Produces, consumed by Task 5: every topic token in `services/` and `libs/` is a `topic.Token`-typed constant, which is what the generator collects by type.

- [ ] **Step 1: Dry-run the codemod and capture the residue**

```bash
cd tools/topicmod && GOWORK=off go build -o /tmp/topicmod ./cmd/topicmod
cd ../.. && /tmp/topicmod -check ./services ./libs 2>&1 | tee /tmp/topicmod-residue.txt
```

Expected: exit 1 (un-migrated sites remain) with a residue list. **Read the residue list in full.** The survey predicts exactly one true residue (`atlas-merchant/.../notification_task.go:52`). If the list names any other site, stop and report it rather than hand-patching past it — an unexpected residue means a rule is wrong, and the fix belongs in Task 3.

- [ ] **Step 2: Apply the codemod and format**

```bash
/tmp/topicmod ./services ./libs
gofmt -l -w services libs
```

- [ ] **Step 3: Hand-edit the four library seams**

`libs/atlas-service/topics.go` — **new file**, the third token shape (raw `os.Getenv` with a literal, survey §J). These four reads are deliberately **optional** — unset means "degrade to legacy single-environment mode", not fatal — so they keep `os.LookupEnv` rather than moving to `EnvProvider`. Declaring the tokens is what puts them in the manifest and what makes `tools/topicguard`'s `raw-env-topic-read` diagnostic pass (it fires on a *literal* argument, not on a `Token`-typed constant):

```go
// Package service's topic tokens. These four reads are intentionally
// optional -- an unset variable degrades to legacy single-environment
// mode rather than failing the process -- so they use os.LookupEnv
// directly rather than topic.EnvProvider, whose contract is now fatal.
package service

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

const (
	EnvEventTopicConfigurationEnvironmentStatus topic.Token = "EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS"
	EnvEventTopicConfigurationTenantStatus      topic.Token = "EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"
	EnvEventTopicConfigurationServiceStatus     topic.Token = "EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS"
)
```

Then rewrite the four call sites to `os.Getenv(string(EnvEventTopicConfiguration...))`, preserving every surrounding warn/skip branch byte-for-byte. (Three constants, not four: `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` is read by both files.)

Retype the producer/outbox seams as listed in the Files block.

- [ ] **Step 4: Build and test every module**

Run: `./tools/test-all-go.sh 2>&1 | tail -60`
Expected: PASS. A compile failure here is a missed retype — find it, and if it is a *pattern* rather than a one-off, add the rule to `tools/topicmod` (Task 2/3 files) rather than hand-patching each site.

- [ ] **Step 5: Confirm the codemod is idempotent and the gate is clean**

```bash
/tmp/topicmod -check ./services ./libs; echo "exit=$?"
```
Expected: `exit=0` — no un-migrated site remains. Any residue reported must be exactly the enumerated hand-edited sites from Step 3, now handled.

- [ ] **Step 6: Commit in two parts**

```bash
git add -A services libs/atlas-kafka libs/atlas-outbox libs/atlas-service
git commit -m "refactor(kafka): retype topic tokens to topic.Token across services and libs"
```

If the working tree cannot be cleanly split (the codemod interleaves both classes in some files), commit once with the message above and record in `context.md` that the split was not achievable; do **not** fabricate a split by staging hunks that do not compile independently.

---

## Task 5: `libs/atlas-kafka/gen` — the scanner, `topics.yaml`, and `policies.yaml`

### Files

- `libs/atlas-kafka/gen/go.mod` — **new file**; module `github.com/Chronicle20/atlas/libs/atlas-kafka/gen`, `go 1.27.0`, requires `golang.org/x/tools v0.49.0` and `gopkg.in/yaml.v3 v3.0.1`. **Not** added to `go.work`.
- `libs/atlas-kafka/gen/main.go` — **new file**; `-check` flag, orchestration, exit codes
- `libs/atlas-kafka/gen/scan.go` — **new file**; the `go/packages` load and `topic.Token` constant collection
- `libs/atlas-kafka/gen/manifest.go` — **new file**; `Manifest`/`Entry` types and YAML emission
- `libs/atlas-kafka/gen/scan_test.go` — **new file**
- `libs/atlas-kafka/gen/policies.yaml` — **new file**; hand-authored
- `libs/atlas-kafka/gen/topics.yaml` — **new file**; generated output
- `go.work` — read-only; the 95 `use` directives the scanner reads
- `services/atlas-kafka-precreate/internal/discover/discover.go` — read-only; lines 18-30 carry the compaction rationale comment to move verbatim into `policies.yaml`

Module root: `libs/atlas-kafka/gen` (`GOWORK=off go build ./...`, `GOWORK=off go test ./...`).

Patterns to copy: `libs/atlas-constants/gen/main.go:44-45` (the `-check` flag) and `libs/atlas-constants/gen/main.go:192-201` (`checkDrift`, a whole-file byte compare returning a "run `go run .` to regenerate" error). `libs/atlas-constants/gen/drift_test.go:23-60` (`TestGeneratedFilesMatchSource`) is the in-process twin of `-check` to mirror.

### Interfaces

- Produces, consumed by Tasks 6, 7, 8:
  - `type Entry struct { Token string; Cleanup string; Packages []string }`
  - `type Manifest struct { Topics []Entry }`
  - `func Scan(repoRoot string) (Manifest, error)` — loads the workspace, collects tokens, applies `policies.yaml`
  - `func (m Manifest) EmitTopicsYAML() []byte`
  - `func checkDrift(path string, want []byte) error`

- [ ] **Step 1: Write the failing test**

`TestScan` in `libs/atlas-kafka/gen/scan_test.go` runs `Scan` against the real repository root (resolved with `git rev-parse --show-toplevel`) and asserts properties rather than a frozen list, so it does not become a second manifest to maintain:

| assertion | expected |
|---|---|
| `len(m.Topics)` | `> 100` (the real count is ~159; a partial load collapses this) |
| entries are sorted by `Token` | `sort.SliceIsSorted` on `Token` |
| every `Entry.Cleanup` | is exactly `"delete"` or `"compact"` |
| tokens with `Cleanup == "compact"` | exactly the three in `policies.yaml`: `EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS`, `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`, `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` |
| `STATUS_TOPIC_CASH_ITEM` present | yes (FR-6.1) |
| `STATUS_EVENT_TOPIC_SKILL_MACRO` present | yes (FR-6.1) |
| every `Entry.Packages` | non-empty and sorted |
| no token from a `_test.go` file | `EVENT_TOPIC_TEST`, `EVENT_TOPIC_FAKE`, `ANY_TOPIC`, `MY_TOPIC`, `RACE_TOPIC`, `TEST_TOPIC` all absent (FR-1.7) |

Plus `TestStalePolicyIsAnError`: call the policy-application step with a policy set containing `EVENT_TOPIC_DOES_NOT_EXIST` and assert the returned error's message contains `EVENT_TOPIC_DOES_NOT_EXIST`.

Plus `TestDrift` (mirroring `libs/atlas-constants/gen/drift_test.go`): `Scan` the repo, `EmitTopicsYAML`, and assert `checkDrift("topics.yaml", want) == nil`.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-kafka/gen && GOWORK=off go test ./... -v`
Expected: FAIL — `Scan` undefined.

- [ ] **Step 3: Implement the scan**

```go
cfg := &packages.Config{
	Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
		packages.NeedTypes | packages.NeedTypesInfo,
	Dir:   repoRoot,
	Tests: false, // FR-1.7: _test.go tokens drop out structurally
	Env:   append(os.Environ(), "GOWORK="+filepath.Join(repoRoot, "go.work")),
}
pkgs, err := packages.Load(cfg, "./services/...", "./libs/...")
```

FR-2.3: after loading, walk `pkgs`; if any package has `len(pkg.Errors) > 0`, return an error naming that package and its first error. A partial load must never produce a manifest.

Collection: for each package, for each name in `pkg.Types.Scope()`, take `*types.Const` values whose `Type()` is the named type `github.com/Chronicle20/atlas/libs/atlas-kafka/topic.Token`; the token string is `constant.StringVal(c.Val())`. Record `pkg.PkgPath` against it. Sort tokens and each `Packages` slice with `sort.Strings`.

Policy: read `policies.yaml` (`compact: [tokens…]`); every listed token must appear in the scan or return an error naming it; entries get `Cleanup: "compact"`, all others `Cleanup: "delete"`. Because policy has exactly one home, FR-2.6's conflicting-policy case is structurally impossible and the generator carries no code for it (design §6).

- [ ] **Step 4: Write `policies.yaml`**

```yaml
# libs/atlas-kafka/gen/policies.yaml -- HAND-AUTHORED. Not generated.
#
# Tokens whose topic must carry cleanup.policy=compact.
#
# Their consumers replay from first-offset at every boot to rebuild
# tenant/service config state and the outbox never re-emits a (topic, key) it
# already delivered, so under the default DELETE cleanup retention empties the
# topic ~7 days after the last config change and every later projection boot has
# nothing to replay. Events are keyed, so compaction retains the latest snapshot
# per key forever.
#   (carried verbatim from services/atlas-kafka-precreate/internal/discover/discover.go:20-27)
compact:
  - EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS
  - EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS
  - EVENT_TOPIC_CONFIGURATION_TENANT_STATUS
```

- [ ] **Step 5: Generate and inspect `topics.yaml`**

Run: `cd libs/atlas-kafka/gen && GOWORK=off go run . && head -20 topics.yaml` and count the entries with `M=topics.yaml; grep -c '^  - token:' "$M"`
Expected: ~159 entries. Header shape:

```yaml
# GENERATED by libs/atlas-kafka/gen -- do not edit.
# Source of truth: topic.Token constants in services/ and libs/.
# Regenerate: tools/gen-topics.sh    Drift check: tools/gen-topics.sh --check
topics:
  - token: COMMAND_TOPIC_ACCOUNT
    cleanup: delete
    packages:
      - atlas-account/kafka/message/account
```

Record the actual count — Task 12's `migration.md` needs it.

- [ ] **Step 6: Run tests and commit**

```bash
cd libs/atlas-kafka/gen && GOWORK=off go test ./... -v
cd ../../.. && git add libs/atlas-kafka/gen
git commit -m "feat(atlas-kafka/gen): scan topic.Token constants into topics.yaml"
```

---

## Task 6: Render `env-configmap.yaml` and `kafka-topics-configmap.yaml`

### Files

- `libs/atlas-kafka/gen/render_configmap.go` — **new file**; the two base-manifest renderers
- `libs/atlas-kafka/gen/splice.go` — **new file**; the shared marker-splice helper
- `libs/atlas-kafka/gen/splice_test.go` — **new file**
- `libs/atlas-kafka/gen/main.go` — wire the two renderers into write and `-check`
- `deploy/k8s/base/env-configmap.yaml` — topic block becomes generated; hand keys move above the markers
- `deploy/k8s/base/kafka-topics-configmap.yaml` — **new file**; wholly generated
- `deploy/k8s/base/kustomization.yaml` — add the new manifest to `resources`

Module root: `libs/atlas-kafka/gen`.

Patterns to copy: `tools/gen-lb-ports.sh:1-28` documents the `# BEGIN generated:<label>` / `# END generated:<label>` marker contract this repo already uses; reuse the label convention, not the bash.

### Interfaces

- Consumes from Task 5: `Manifest`, `Entry`, `checkDrift`.
- Produces, consumed by Task 7:
  - `func Splice(existing []byte, beginMarker, endMarker string, block []byte) ([]byte, error)` — replaces the region between the markers, preserving every byte outside it and the file's existing line endings; errors if either marker is missing or they are out of order.
  - `func (m Manifest) EmitEnvConfigMapBlock() []byte`
  - `func (m Manifest) EmitTopicsConfigMap() []byte`

- [ ] **Step 1: Write the failing test**

`TestSplice` in `libs/atlas-kafka/gen/splice_test.go`, table-driven over in-memory byte slices (no fixture files):

| subtest | input | block | expect |
|---|---|---|---|
| `replaces marked region` | `"a\n# B\nold\n# E\nz\n"` | `"new\n"` | `"a\n# B\nnew\n# E\nz\n"` |
| `preserves CRLF outside markers` | `"a\r\n# B\r\nold\r\n# E\r\nz\r\n"` | `"new\r\n"` | `"a\r\n# B\r\nnew\r\n# E\r\nz\r\n"` |
| `empty block` | `"a\n# B\nold\n# E\n"` | `""` | `"a\n# B\n# E\n"` |
| `missing begin marker` | `"a\nold\n# E\n"` | `"new\n"` | error containing `# B` |
| `missing end marker` | `"a\n# B\nold\n"` | `"new\n"` | error containing `# E` |
| `end before begin` | `"# E\nx\n# B\n"` | `"new\n"` | error containing `out of order` |

Markers in the table are `# B` (begin) and `# E` (end), passed as arguments.

Plus `TestEmitEnvConfigMapBlock`: build a two-entry `Manifest` (`COMMAND_TOPIC_A`, `EVENT_TOPIC_B`) and assert the emitted block is exactly:

```
  COMMAND_TOPIC_A: "COMMAND_TOPIC_A"
  EVENT_TOPIC_B: "EVENT_TOPIC_B"
```

Plus `TestEmitTopicsConfigMap`: assert the same two-entry manifest emits a ConfigMap named `atlas-kafka-topics` with a `topics.yaml` key whose value carries `token`/`cleanup` pairs and **no** `packages:` key (design §5 — provenance is stripped from the mounted copy), and with the `argocd.argoproj.io/sync-wave: "-1"` annotation.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-kafka/gen && GOWORK=off go test ./... -run 'TestSplice|TestEmit' -v`
Expected: FAIL — `Splice` undefined.

- [ ] **Step 3: Restructure `env-configmap.yaml` by hand, once**

The file today is globally alphabetical, which puts the topics in two non-contiguous regions (21–102 and 105–196) split by `DB_HOST`/`DB_PORT` at 103–104. Move the eleven hand-written keys (`ATLAS_ENVIRONMENT` **with its 12-line rationale comment at lines 6-17 preserved byte-for-byte**, `BASE_SERVICE_URL`, `BOOTSTRAP_SERVERS`, `DB_HOST`, `DB_PORT`, `LEACH_INTERVAL`, `LEVEL_INTERVAL`, `REDIS_URL`, `REST_PORT`, `TRACE_ENDPOINT`, `TRACE_SAMPLING_RATIO`, `USE_ENFORCE_MOB_LEVEL_RANGE`) above the markers, and place one generated block at the tail of `data:`:

```yaml
  USE_ENFORCE_MOB_LEVEL_RANGE: "true"
  # BEGIN generated:topics (libs/atlas-kafka/gen -- run tools/gen-topics.sh)
  # END generated:topics
```

- [ ] **Step 4: Implement the renderers and generate**

`EmitEnvConfigMapBlock` writes `  TOKEN: "TOKEN"` per entry in manifest order (identity mapping — the per-environment suffix is the overlays' job, FR-3.1).

`EmitTopicsConfigMap` writes the whole file:

```yaml
# GENERATED by libs/atlas-kafka/gen -- do not edit. Run tools/gen-topics.sh.
#
# The topic set atlas-kafka-precreate creates, mounted into that Job at
# /etc/atlas/topics/topics.yaml. Carries only the token and its cleanup
# policy; the resolved per-environment topic name comes from atlas-env,
# which the overlays generate from the same manifest.
apiVersion: v1
kind: ConfigMap
metadata:
  name: atlas-kafka-topics
  annotations:
    # Wave -1: this ConfigMap is mounted by the wave-0 atlas-kafka-precreate
    # Job. ArgoCD applies same-wave resources before health-checking any of
    # them and a configMap volume resolves at pod start, so a same-wave
    # ConfigMap races the Job into ContainerCreating.
    argocd.argoproj.io/sync-wave: "-1"
data:
  topics.yaml: |
    topics:
      - token: COMMAND_TOPIC_ACCOUNT
        cleanup: delete
```

Add `kafka-topics-configmap.yaml` to `deploy/k8s/base/kustomization.yaml`'s `resources` list, in its existing alphabetical position.

Run: `cd libs/atlas-kafka/gen && GOWORK=off go run . && cd ../../.. && git diff --stat deploy/k8s/base/`

- [ ] **Step 5: Verify the orphan removal and the two additions**

```bash
git diff deploy/k8s/base/env-configmap.yaml | grep '^-  [A-Z]' | wc -l   # removed keys
git diff deploy/k8s/base/env-configmap.yaml | grep '^+  STATUS_'          # the two FR-6.1 additions
```
Expected: the removed-key count equals the orphan count from design §1 (17) plus any keys merely relocated — inspect the diff rather than trusting the count alone, and **record the exact removed key list**; Task 12's `migration.md` requires it and it must not be invented.

- [ ] **Step 6: Run tests and commit**

```bash
cd libs/atlas-kafka/gen && GOWORK=off go test ./... && GOWORK=off go run . -check; echo "check exit=$?"
cd ../../.. && git add libs/atlas-kafka/gen deploy/k8s/base
git commit -m "feat(atlas-kafka/gen): render env-configmap topic block and the topics ConfigMap"
```
`check exit=0` is required before committing.

---

## Task 7: Render the three overlays and `deploy/compose/.env.example`

### Files

- `libs/atlas-kafka/gen/render_overlays.go` — **new file**; the suffix table and the four remaining renderers
- `libs/atlas-kafka/gen/render_overlays_test.go` — **new file**
- `libs/atlas-kafka/gen/main.go` — wire the renderers into write and `-check`
- `deploy/k8s/overlays/main/kustomization.yaml` — topic literals (lines 60-233) become a marked generated block
- `deploy/k8s/overlays/pr/kustomization.yaml` — same (lines 180-353)
- `deploy/k8s/overlays/pr-sparse/kustomization.yaml` — same (lines 343-516); the comment at line 288 explaining the historical unsuffixed-topic bug is **kept**
- `deploy/compose/.env.example` — topic block (lines 20-112) becomes generated
- `deploy/k8s/overlays/pr/scripts/gen-topic-config.sh` — **deleted**; the generator subsumes it (FR-3.4)
- `tools/pr-sparse-mirror-guard.sh` — read-only; confirmed its `MIRRORS` array does not list `kustomization.yaml`
- `tools/overlay-env-guard.sh` — read-only; asserts on `ATLAS_ENVIRONMENT` and ingress wiring, not topics

Module root: `libs/atlas-kafka/gen`.

### Interfaces

- Consumes from Task 6: `Splice`, `Manifest`.
- Produces: `func (m Manifest) EmitOverlayBlock(suffix string) []byte`, `func (m Manifest) EmitComposeBlock() []byte`.

- [ ] **Step 1: Write the failing test**

`TestEmitOverlayBlock`, table-driven on the suffix, using a two-entry manifest (`COMMAND_TOPIC_A`, `EVENT_TOPIC_B`):

| overlay | suffix argument | expected block (exact) |
|---|---|---|
| `main` | `-main` | `      - COMMAND_TOPIC_A=COMMAND_TOPIC_A-main\n      - EVENT_TOPIC_B=EVENT_TOPIC_B-main\n` |
| `pr` | `-PLACEHOLDER_ATLAS_ENV` | `      - COMMAND_TOPIC_A=COMMAND_TOPIC_A-PLACEHOLDER_ATLAS_ENV\n      - EVENT_TOPIC_B=EVENT_TOPIC_B-PLACEHOLDER_ATLAS_ENV\n` |
| `pr-sparse` | `-PLACEHOLDER_BASELINE_ENVIRONMENT` | `      - COMMAND_TOPIC_A=COMMAND_TOPIC_A-PLACEHOLDER_BASELINE_ENVIRONMENT\n      - EVENT_TOPIC_B=EVENT_TOPIC_B-PLACEHOLDER_BASELINE_ENVIRONMENT\n` |

Six-space indent is what the existing literals use (`deploy/k8s/overlays/main/kustomization.yaml:60`).

`TestEmitComposeBlock` on the same manifest expects exactly `COMMAND_TOPIC_A=COMMAND_TOPIC_A\nEVENT_TOPIC_B=EVENT_TOPIC_B\n` (shell form, no indent, no quotes).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-kafka/gen && GOWORK=off go test ./... -run 'TestEmitOverlay|TestEmitCompose' -v`
Expected: FAIL — `EmitOverlayBlock` undefined.

- [ ] **Step 3: Add the markers to the four files by hand**

Wrap each existing topic block with `# BEGIN generated:topics (libs/atlas-kafka/gen -- run tools/gen-topics.sh)` / `# END generated:topics` at the block's own indentation (six spaces inside the `literals:` sequences, column 0 in `.env.example`).

- [ ] **Step 4: Implement the renderers with the suffix table**

The suffix table carries the rationale from the deleted `gen-topic-config.sh` verbatim — this comment is the only record of the 2026-08-20 `atlas-login` crash-loop and must survive the deletion:

```go
// overlaySuffixes maps each overlay to the suffix its topic names carry.
//
//   -PLACEHOLDER_ATLAS_ENV          -- overlays/pr, isolated mode: this
//     environment gets its own topics, suffixed with its own env token.
//   -PLACEHOLDER_BASELINE_ENVIRONMENT -- overlays/pr-sparse, sparse mode:
//     this environment shares the BASELINE's topics, so it must name them
//     the way the baseline names them. Note this is not the same as "no
//     suffix": the baseline overlay suffixes every topic with its own
//     environment id (`-main` today), so an unsuffixed name addresses a
//     topic nobody publishes to. That was the atlas-login crash-loop of
//     2026-08-20 -- see docs/tasks/task-232-sparse-ephemeral-environments/
//     bug-sparse-baseline-scoping.md.
var overlaySuffixes = map[string]string{
	"main":      "-main",
	"pr":        "-PLACEHOLDER_ATLAS_ENV",
	"pr-sparse": "-PLACEHOLDER_BASELINE_ENVIRONMENT",
}
```

Note what disappears with the script: its `test("^(COMMAND|EVENT)_TOPIC_")` selector, the mechanism that silently dropped the two `STATUS_*` tokens. The manifest has no prefix filter at all.

- [ ] **Step 5: Generate, delete the script, and render the overlays**

```bash
cd libs/atlas-kafka/gen && GOWORK=off go run . && cd ../../..
git rm deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
grep -rn 'gen-topic-config' --include='*.yaml' --include='*.sh' --include='*.yml' . | grep -v '^./docs/'
```
The `grep` must return nothing outside `docs/`. If a workflow or script still calls it, update that caller in this task — a dangling reference is not a follow-up.

- [ ] **Step 6: Assert every rendered overlay's key set equals the manifest**

```bash
for o in main pr pr-sparse; do
  kustomize build "deploy/k8s/overlays/$o" > "/tmp/render-$o.yaml" || echo "BUILD FAILED: $o"
done
MANIFEST=libs/atlas-kafka/gen/topics.yaml; grep -c '^  - token:' "$MANIFEST"
for o in main pr pr-sparse; do
  echo -n "$o rendered topic keys: "
  grep -oE '^\s+[A-Z0-9_]*TOPIC[A-Z0-9_]*:' "/tmp/render-$o.yaml" | sort -u | wc -l
done
```
Expected: all three `kustomize build`s succeed and each rendered count equals the manifest token count (FR-3.5, PRD acceptance criterion 8).

- [ ] **Step 7: Run tests and commit**

```bash
cd libs/atlas-kafka/gen && GOWORK=off go test ./... && GOWORK=off go run . -check; echo "check exit=$?"
cd ../../.. && git add -A libs/atlas-kafka/gen deploy
git commit -m "feat(atlas-kafka/gen): generate overlay topic literals and compose env example"
```

---

## Task 8: `tools/gen-topics.sh` and the `verify.sh` drift step

### Files

- `tools/gen-topics.sh` — **new file**; thin wrapper, `chmod +x`
- `tools/gen-topics_test.sh` — **new file**; bats suite
- `tools/verify.sh` — add the drift step
- `tools/gen-lb-ports.sh` — read-only; the wrapper shape to mirror (header comment + `--check` handling at lines 1-28)
- `tools/gen-tenant-tables_test.sh` — read-only; the bats suite shape to mirror

Module root: n/a (bash).

Patterns to copy: `tools/verify.sh:566-572` (the `touched`/`step`/`skip` block for `gen-lb-ports.sh --check`); `tools/verify.sh:575-580` (the `gen-tenant-tables` block, which also runs the generator's own test suite).

- [ ] **Step 1: Write the failing test**

`tools/gen-topics_test.sh`, bats, three cases:

| test name | action | expect |
|---|---|---|
| `gen-topics.sh --check exits 0 on a clean tree` | run `tools/gen-topics.sh --check` | exit 0 |
| `gen-topics.sh --check exits 1 on a dirty manifest` | append `  - token: EVENT_TOPIC_FABRICATED\n    cleanup: delete\n` to a temp copy... see note | exit 1, stderr names `topics.yaml` |
| `gen-topics.sh --check writes no files` | snapshot `git status --porcelain` before and after `--check` | identical |

For case 2, mutate the real `libs/atlas-kafka/gen/topics.yaml`, run `--check`, then restore it with `git checkout -- libs/atlas-kafka/gen/topics.yaml` in a `teardown`. Do not fabricate a temp repo.

- [ ] **Step 2: Run test to verify it fails**

Run: `./tools/gen-topics_test.sh`
Expected: FAIL — `tools/gen-topics.sh: No such file or directory`.

- [ ] **Step 3: Write the wrapper**

```bash
#!/usr/bin/env bash
# Regenerate every artifact derived from the topic.Token constants in
# services/ and libs/: libs/atlas-kafka/gen/topics.yaml, the marked topic
# block in deploy/k8s/base/env-configmap.yaml, deploy/k8s/base/
# kafka-topics-configmap.yaml, the marked literals in all three overlay
# kustomizations, and deploy/compose/.env.example. task-276 FR-2/FR-3.
#
#   gen-topics.sh           rewrite the marker blocks in place
#   gen-topics.sh --check   exit 1 with a diff on drift; writes nothing
#
# The generator is its own module and is deliberately outside go.work, so
# it is invoked with GOWORK=off (same posture as tools/atlasguards).
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT/libs/atlas-kafka/gen"
GOWORK=off exec go run . "$@"
```

`chmod +x tools/gen-topics.sh`.

- [ ] **Step 4: Wire `verify.sh`**

Insert after the `gen-tenant-tables` block (`tools/verify.sh:575-580`):

```bash
if [ "$ALL" -eq 1 ] || touched '^(libs/atlas-kafka/gen/|tools/gen-topics(_test)?\.sh|deploy/k8s/base/env-configmap\.yaml|deploy/k8s/base/kafka-topics-configmap\.yaml|deploy/k8s/overlays/[a-z-]+/kustomization\.yaml|deploy/compose/\.env\.example)'; then
    step "topic manifest drift"      ./tools/gen-topics.sh --check
    step "topic generator tests"     bash -c 'cd libs/atlas-kafka/gen && GOWORK=off go test ./...'
else
    skip "topic manifest drift (no manifest or topic deploy surface changed)"
fi
```

The trigger is deliberately **not** `\.go$` (design §4c): `tools/topicguard`'s `token-not-in-manifest` diagnostic (Task 9) covers a token added in Go without regeneration, at ~zero marginal cost inside the vet sweep that already runs on every `.go` change. FR-5.2's escape clause is satisfied by that pre-check.

- [ ] **Step 5: Run tests**

```bash
./tools/gen-topics_test.sh
time ./tools/gen-topics.sh --check
```
Expected: bats PASS; `--check` exits 0. Record the wall time — NFR §8 budgets 30s warm. If it exceeds that, report the number rather than silently accepting it.

- [ ] **Step 6: Commit**

```bash
git add tools/gen-topics.sh tools/gen-topics_test.sh tools/verify.sh
git commit -m "feat(verify): add the topic manifest drift gate"
```

---

## Task 9: `tools/topicguard` analyzer

### Files

- `tools/topicguard/go.mod` — **new file**; module `github.com/Chronicle20/atlas/tools/topicguard`, `go 1.27.0`, `golang.org/x/tools v0.49.0`
- `tools/topicguard/analyzer.go` — **new file**; the three diagnostics
- `tools/topicguard/analyzer_test.go` — **new file**; `analysistest`
- `tools/topicguard/testdata/src/atlas-example/bareliteral/x.go` — **new file**
- `tools/topicguard/testdata/src/atlas-example/rawenv/x.go` — **new file**
- `tools/topicguard/testdata/src/atlas-example/declared/x.go` — **new file**; the passing shape
- `tools/topicguard/testdata/src/github.com/Chronicle20/atlas/libs/atlas-kafka/topic/topic.go` — **new file**; a stub `type Token string` for the fixtures to import
- `tools/atlasguards/guards.go` — register the sixth analyzer in `Services()` (lines 46-55) and `Libraries()` (lines 57-64), add the import (lines 36-44), extend the doc-comment table (lines 15-19)
- `tools/atlasguards/go.mod` — add the `require` + `replace` pair, mirroring the five existing pairs
- `tools/go-analyzer-guards.sh` — add `"$ROOT/tools/topicguard"` to `GUARD_SRCS` (lines 51-58) and `topicguard` to `SELFTEST_GUARDS` (line 62)
- `libs/atlas-kafka/gen/topics.yaml` — **new file** in Task 5; read-only here — the manifest diagnostic 3 reads
- `tools/scopeguard/analyzer.go` — read-only; the analyzer shape
- `tools/scopeguard/allowlist.go` — read-only; the external-data shape

Module roots: `tools/topicguard` and `tools/atlasguards` (both `GOWORK=off`).

Patterns to copy: `tools/scopeguard/analyzer.go:98-103` (the `*analysis.Analyzer` declaration with `inspect.Analyzer` as its sole `Requires`, read back via `pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)` at `analyzer.go:107`); `tools/scopeguard/analyzer_test.go:1-20` (`TestAnalyzer` using `analysistest.Run` with `testdata/src/<pkg>` fixtures).

**Manifest access.** The other five guards embed their data with `go:embed` (`tools/scopeguard/allowlist.go:9-13`), but `topics.yaml` lives outside this module and cannot be embedded across a module boundary. Use a `flag.String("topicguard.manifest", …)` on the `Analyzer.Flags` set, defaulting to the path found by walking up from the analyzed package's directory to the repo root and appending `libs/atlas-kafka/gen/topics.yaml`. If the manifest cannot be read, diagnostic 3 is **skipped silently** (diagnostics 1 and 2 still run) — an analyzer that hard-fails on a missing manifest would break `analysistest`, which runs in a synthetic GOPATH.

- [ ] **Step 1: Write the failing test**

`TestAnalyzer` in `tools/topicguard/analyzer_test.go`, `analysistest.Run(t, analysistest.TestData(), Analyzer, "atlas-example/bareliteral", "atlas-example/rawenv", "atlas-example/declared")`. Expectations are `// want` comments in the fixtures.

`testdata/src/atlas-example/bareliteral/x.go`:

```go
package bareliteral

import "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"

func put(t topic.Token) {}

const untyped = "EVENT_TOPIC_UNTYPED"

func f() {
	put("EVENT_TOPIC_LITERAL") // want `bare topic literal "EVENT_TOPIC_LITERAL" reaching a topic.Token parameter; declare it as a topic.Token constant`
	put(untyped)               // want `bare topic literal "EVENT_TOPIC_UNTYPED" reaching a topic.Token parameter; declare it as a topic.Token constant`
}
```

`testdata/src/atlas-example/rawenv/x.go`:

```go
package rawenv

import "os"

func f() string {
	return os.Getenv("EVENT_TOPIC_RAW") // want `raw environment read of topic token "EVENT_TOPIC_RAW"; reference a topic.Token constant instead`
}

func g() (string, bool) {
	return os.LookupEnv("COMMAND_TOPIC_RAW") // want `raw environment read of topic token "COMMAND_TOPIC_RAW"; reference a topic.Token constant instead`
}

func ok() string {
	return os.Getenv("REST_PORT")
}
```

`testdata/src/atlas-example/declared/x.go` — the passing shape: a `topic.Token`-typed constant passed to `put`, plus `os.Getenv(string(EnvSomeTopic))`. No `// want` comments; any diagnostic here fails the test.

Diagnostic 3 is exercised by a separate unit test, `TestTokenNotInManifest`, which calls the manifest-membership helper directly with an in-memory token set rather than through `analysistest` (the fixture GOPATH has no repo root to walk to):

| case | manifest set | token | expect diagnostic |
|---|---|---|---|
| present | `{COMMAND_TOPIC_A}` | `COMMAND_TOPIC_A` | none |
| absent | `{COMMAND_TOPIC_A}` | `EVENT_TOPIC_B` | `topic token "EVENT_TOPIC_B" is not in libs/atlas-kafka/gen/topics.yaml; run tools/gen-topics.sh` |
| manifest unreadable | `nil` | `EVENT_TOPIC_B` | none (skipped) |

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/topicguard && GOWORK=off go test ./... -v`
Expected: FAIL — `Analyzer` undefined.

- [ ] **Step 3: Implement the three diagnostics**

1. **`bare-token-literal`** — at every `*ast.CallExpr`, for each argument whose parameter type (from `pass.TypesInfo`) is the named type `…/atlas-kafka/topic.Token`: if the argument's own type is *untyped* string (a literal, or an untyped constant) rather than a reference to a `topic.Token`-typed constant, report. Closes F1/FR-1.4 — a defined type alone cannot, because Go converts untyped string constants implicitly.
2. **`raw-env-topic-read`** — `os.Getenv` / `os.LookupEnv` whose sole argument is a `*ast.BasicLit` string matching `^[A-Z0-9_]*TOPIC[A-Z0-9_]*$`, outside package `…/atlas-kafka/topic`. A `string(SomeConst)` conversion argument does not match and is the sanctioned form.
3. **`token-not-in-manifest`** — every `*types.Const` of type `topic.Token` in the analyzed package whose value is absent from the loaded manifest.

- [ ] **Step 4: Register it in the shared sweep**

`tools/atlasguards/guards.go`: add `topicguard.Analyzer` to **both** `Services()` and `Libraries()` — the raw-`os.Getenv` sites live in `libs/atlas-service`, so a services-only registration would miss them (design §4). Add the `require`/`replace` pair to `tools/atlasguards/go.mod`. Add the module dir to `GUARD_SRCS` and `topicguard` to `SELFTEST_GUARDS` in `tools/go-analyzer-guards.sh`.

- [ ] **Step 5: Run the analyzer over the real tree**

```bash
cd tools/topicguard && GOWORK=off go test ./... -v && cd ../..
./tools/go-analyzer-guards.sh
```
Expected: the guard script exits 0. It will not, if Task 4 left a site behind — that is the point. Fix the site (or the codemod rule), do not add an allowlist.

- [ ] **Step 6: Commit**

```bash
git add tools/topicguard tools/atlasguards tools/go-analyzer-guards.sh
git commit -m "feat(topicguard): gate bare topic literals, raw env reads, and manifest drift"
```

---

## Task 10: `atlas-kafka-precreate` manifest package

### Files

- `services/atlas-kafka-precreate/internal/manifest/manifest.go` — **new file**; `Manifest`, `Entry`, `Parse`, `Resolve`, `Load`
- `services/atlas-kafka-precreate/internal/manifest/manifest_test.go` — **new file**
- `services/atlas-kafka-precreate/go.mod` — add `gopkg.in/yaml.v3 v3.0.1` if not already required
- `services/atlas-kafka-precreate/internal/discover/discover.go` — read-only in this task; `Topics` is the shared vocabulary `Resolve` returns

Module root: `services/atlas-kafka-precreate` (module path `atlas.com/kafka-precreate`; it **is** in `go.work` at `go.work:53`, so plain `go test ./...` works here).

Patterns to copy: `services/atlas-kafka-precreate/internal/discover/discover_test.go:8-88` (`TestFromEnviron`) — the struct-literal table with `wantPlain`/`wantCompact` slices is the exact shape to reuse, including the "compaction wins on collision" and "duplicates collapsed" cases.

### Interfaces

- Produces, consumed by Task 11:
  - `type Entry struct { Token string \`yaml:"token"\`; Cleanup string \`yaml:"cleanup"\` }`
  - `type Manifest struct { Topics []Entry \`yaml:"topics"\` }`
  - `func Parse(data []byte) (Manifest, error)`
  - `func Resolve(m Manifest, look func(string) (string, bool)) (discover.Topics, error)`
  - `func Load(path string) (Manifest, error)`

- [ ] **Step 1: Write the failing test**

`TestParse` — table over raw YAML bytes:

| case | input | expect |
|---|---|---|
| `well formed` | `topics:\n  - token: COMMAND_TOPIC_A\n    cleanup: delete\n` | one entry, `{COMMAND_TOPIC_A, delete}` |
| `malformed yaml` | `topics: [` | error containing `parsing topic manifest` |
| `empty document` | `""` | error containing `topic manifest is empty` |
| `no topics key` | `other: 1\n` | error containing `topic manifest is empty` |
| `unknown cleanup value` | `topics:\n  - token: A\n    cleanup: squash\n` | error containing `squash` |
| `missing cleanup` | `topics:\n  - token: A\n` | error containing `A` |

`TestResolve` — table over a `Manifest` plus a fake `look` function backed by a `map[string]string`:

| case | manifest tokens (cleanup) | env | wantPlain | wantCompact | wantErr |
|---|---|---|---|---|---|
| `plain and compact` | `A`(delete), `B`(compact) | `A=a`, `B=b` | `["a"]` | `["b"]` | nil |
| `duplicates collapsed` | `A`(delete), `B`(delete) | `A=shared`, `B=shared` | `["shared"]` | `[]` | nil |
| `compaction wins on collision` | `A`(delete), `B`(compact) | `A=both`, `B=both` | `[]` | `["both"]` | nil |
| `sorted output` | `A`,`B`,`C` all delete | `A=topic_b`, `B=topic-a`, `C=topicZ` | `["topic-a","topicZ","topic_b"]` | `[]` | nil |
| `unresolved token is fatal` | `A`(delete), `B`(delete) | `A=a` only | — | — | error is exactly `topic manifest token [B] has no value in the environment` |
| `empty value is fatal` | `A`(delete) | `A=` | — | — | error is exactly `topic manifest token [A] has no value in the environment` |
| `first unresolved by sort order` | `Z`(delete), `A`(delete) | neither set | — | — | error names `A`, not `Z` (FR-4.4: walk in sorted order, report the first) |

The `sorted output` row copies `TestFromEnviron`'s "underscore vs hyphen ordering is byte order" case verbatim so the new package provably preserves the old ordering contract. `wantPlain`/`wantCompact` are never nil (FR-4.6 preserves `discover.Topics`' "never nil" invariant).

`TestLoad` — two cases: a temp file written with the `well formed` document round-trips to the same `Manifest`; a nonexistent path returns an error containing that path.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd services/atlas-kafka-precreate && go test ./internal/manifest/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement the package**

`Resolve` walks `m.Topics` sorted by `Token`, calls `look(entry.Token)`, and on `!ok || value == ""` returns the error above naming the token. Otherwise it classifies into two sets and returns `discover.Topics{Plain: sortedKeys(plain), Compact: sortedKeys(compact)}` with the compaction-wins de-duplication `FromEnviron` implements today (a name in both sets ends up only in `Compact`).

`Parse` validates that `Cleanup` is exactly `"delete"` or `"compact"`.

- [ ] **Step 4: Run tests**

Run: `cd services/atlas-kafka-precreate && go test ./... -v 2>&1 | tail -30`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-kafka-precreate
git commit -m "feat(kafka-precreate): parse and resolve the mounted topic manifest"
```

---

## Task 11: Rewire `atlas-kafka-precreate` and mount the ConfigMap

### Files

- `services/atlas-kafka-precreate/main.go` — replace the `discover.FromEnviron` call at line 57 with the manifest load/resolve
- `services/atlas-kafka-precreate/internal/discover/discover.go` — delete `FromEnviron`, `commandPrefix`, `eventPrefix`, `compactVars`, and any helper only they used; **keep** `Topics`, `Union`, `sortedKeys`, `Groups`, `StateIsSeedable` (FR-4.7)
- `services/atlas-kafka-precreate/internal/discover/discover_test.go` — delete `TestFromEnviron` (lines 8-88); keep the `Groups` and `StateIsSeedable` tests untouched
- `deploy/k8s/base/atlas-kafka-precreate.yaml` — add `volumes:` + `volumeMounts:`; rewrite the header comment's discovery sentence
- `services/atlas-kafka-precreate/README.md` — update the discovery description if it names `FromEnviron` or the prefix scrape

Module root: `services/atlas-kafka-precreate`.

### Interfaces

- Consumes from Task 10: `manifest.Load`, `manifest.Resolve`, `manifest.Manifest`.
- Consumes from Task 6: the ConfigMap `atlas-kafka-topics` with key `topics.yaml`.

- [ ] **Step 1: Rewire `main.go`**

Replace `t := discover.FromEnviron(os.Environ())` (line 57) with:

```go
	// Phase A: discover. The topic set is the code-derived manifest mounted
	// from the atlas-kafka-topics ConfigMap; names still resolve through
	// atlas-env because they carry the per-environment suffix the manifest
	// deliberately does not encode (task-276 FR-4.3).
	path := os.Getenv("KAFKA_TOPIC_MANIFEST_PATH")
	if path == "" {
		path = "/etc/atlas/topics/topics.yaml"
	}
	m, err := manifest.Load(path)
	if err != nil {
		return fmt.Errorf("loading topic manifest from %s: %w", path, err)
	}
	t, err := manifest.Resolve(m, os.LookupEnv)
	if err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"phase":    "discover",
		"manifest": path,
		"tokens":   len(m.Topics),
		"compact":  len(t.Compact),
	}).Info("topic manifest loaded")
```

`KAFKA_TOPIC_MANIFEST_PATH` is read with `os.Getenv` and a hardcoded default. It is **not** added to the Job's container `env:` — see Step 3.

The later `ensureResult, err := topics.Ensure(...)` at line 62 must become `ensureResult, err = topics.Ensure(...)` since `err` is now already declared. Everything downstream (`Settle`, `EndOffsets`, `groups.Seed`, `Verify`, the exit-code contract) is untouched (FR-4.6).

- [ ] **Step 2: Delete the prefix scrape**

Remove `FromEnviron`, `commandPrefix`, `eventPrefix`, and `compactVars` from `discover.go` (lines 12-30 and the `FromEnviron` body), and `TestFromEnviron` from `discover_test.go`. The compaction rationale comment that sat above `compactVars` now lives in `libs/atlas-kafka/gen/policies.yaml` (Task 5) — confirm it is there before deleting it here.

- [ ] **Step 3: Mount the ConfigMap**

In `deploy/k8s/base/atlas-kafka-precreate.yaml`, append to the container and pod spec (the file ends at the `envFrom:` block, line 61):

```yaml
          volumeMounts:
            - name: topic-manifest
              mountPath: /etc/atlas/topics
              readOnly: true
      volumes:
        - name: topic-manifest
          configMap:
            name: atlas-kafka-topics
```

**The container must still carry no `env:` key.** The file's own comment (lines 55-60) records why: `.github/workflows/pr-validation.yml`'s JSON-6902 patch does `op: add` on `/spec/template/spec/containers/0/env` to inject `KAFKA_CONSUMER_GROUP`, and `op: add` on an already-present key REPLACES it. Neither `volumes:` nor `volumeMounts:` is touched by that patch.

Rewrite the header comment's mechanism sentence (currently "reads its own envFrom-injected atlas-env ConfigMap, picks out every `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` value") to describe the manifest: it reads the code-derived token list from the mounted `atlas-kafka-topics` ConfigMap and resolves each token's per-environment name through the envFrom-injected `atlas-env`.

- [ ] **Step 4: Verify the constraint and the deletions**

```bash
cd services/atlas-kafka-precreate && go build ./... && go test ./... -v 2>&1 | tail -20 && cd ../..
grep -rn 'FromEnviron\|commandPrefix\|eventPrefix\|compactVars' services/atlas-kafka-precreate/   # expect: no output
awk '/name: kafka-precreate/,0' deploy/k8s/base/atlas-kafka-precreate.yaml | grep -n '^\s*env:'   # expect: no output
kustomize build deploy/k8s/base > /dev/null && echo "base builds"
```
All four expectations must hold (PRD acceptance criteria 9, 10, 11).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-kafka-precreate deploy/k8s/base/atlas-kafka-precreate.yaml
git commit -m "feat(kafka-precreate): read the mounted topic manifest instead of scraping the environment"
```

---

## Task 12: `migration.md` and the full verification gate

### Files

- `docs/tasks/task-276-kafka-topic-manifest/migration.md` — **new file** (FR-6.3)
- `docs/tasks/task-276-kafka-topic-manifest/plan.md` — read-only
- `tools/verify.sh` — read-only; the gate to run

- [ ] **Step 1: Recover the exact before/after topic sets from git**

```bash
BASE=$(git merge-base HEAD main)
git show "$BASE:deploy/k8s/base/env-configmap.yaml" \
  | grep -oE '^  [A-Z0-9_]*TOPIC[A-Z0-9_]*:' | tr -d ' :' | sort > /tmp/topics-before.txt
grep -oE '^  - token: .*' libs/atlas-kafka/gen/topics.yaml | awk '{print $3}' | sort > /tmp/topics-after.txt
wc -l /tmp/topics-before.txt /tmp/topics-after.txt
echo "=== removed (orphans) ==="; comm -23 /tmp/topics-before.txt /tmp/topics-after.txt
echo "=== added ==="; comm -13 /tmp/topics-before.txt /tmp/topics-after.txt
```

Every name in `migration.md` comes from this command's output. **Do not transcribe a token from memory or from design.md's prose** — the design gives counts (17 removed, 2 added), not names.

- [ ] **Step 2: Write `migration.md`**

Sections, in order:

1. **Counts** — before, after, removed, added, each with the `wc -l`/`comm` output that produced it.
2. **Removed (orphan `env-configmap.yaml` keys with no Go declaration)** — the `comm -23` list verbatim. Note per design §12 that all 17 appear in `deploy/compose/.env.example` (regenerated here) and 4 additionally in historical `docs/tasks/**` prose only; no script, Job, or external tool references any of them (FR-6.2). Topics already created in live environments are left in place, unused.
3. **Added** — the `comm -13` list verbatim, expected to be `STATUS_TOPIC_CASH_ITEM` and `STATUS_EVENT_TOPIC_SKILL_MACRO` (FR-6.1). For each, the rendered name change per environment: previously unsuffixed (the deleted `gen-topic-config.sh`'s `^(COMMAND|EVENT)_TOPIC_` selector dropped them, so no overlay ever suffixed them); now `<TOKEN>-main` on `main`, `<TOKEN>-<env>` on `pr`, `<TOKEN>-<baseline>` on `pr-sparse`.
4. **Consumer impact of the two renames** — `STATUS_TOPIC_CASH_ITEM` has **no consumer anywhere in the repo**: one declaration (`services/atlas-cashshop/atlas.com/cashshop/kafka/message/item/kafka.go:5`), four producer sites in `cashshop/inventory/asset/processor.go`, no `InitConsumers`/`InitHandlers`/`rf(` registration. `STATUS_EVENT_TOPIC_SKILL_MACRO` is consumed by `services/atlas-channel/.../kafka/consumer/macro/consumer.go:29`, which registers with `consumer.SetStartOffset(kafka.LastOffset)` and therefore never replays history — repointing it loses nothing it would have read.
5. **In-flight outbox window** — `libs/atlas-outbox` persists the **resolved** name into `outbox_entries.topic` (`entity.go:11`, written at `bridge.go:42`), so unsent rows enqueued before the change still name the old topic. `TopicWriterPool` sets `AllowAutoTopicCreation: true` (`publisher.go:86`), so they drain rather than erroring. Neither renamed token is produced through the outbox bridge today (cashshop's four sites use `mb.Put` under the direct `message.Emit` path), so the window is empty in practice. Recorded, not mitigated.
6. **Behaviour change** — `topic.EnvProvider` no longer falls back to the token when a variable is unset; it errors. Consumer registration paths fatal at boot on an unresolved token; handler registration paths propagate. This is what makes an unmanaged token visible instead of silently addressing a topic nobody publishes to.

- [ ] **Step 3: Run the full gate**

Run: `./tools/verify.sh 2>&1 | tail -60`
Expected: exit 0, flagless. `--quick`/`--no-docker` do **not** count (CLAUDE.md "Done means verified"). If any step fails, fix it in the task that owns it rather than in this one.

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-276-kafka-topic-manifest/migration.md
git commit -m "docs(task-276): record the topic manifest before/after migration"
```

---

## Spec coverage

| Requirement | Task |
|---|---|
| FR-1.1 `type Token string` | 1 |
| FR-1.2 every token declared `topic.Token` | 2 (rule), 4 (applied) |
| FR-1.3 token-taking functions retyped (minus `consumer.NewConfig` — design §3) | 4 |
| FR-1.4 no bare literal reaches a token parameter | 9 (`bare-token-literal`) |
| FR-1.5 `EnvProvider` errors instead of falling back | 1 |
| FR-1.6 every discarded error handled | 3 (rules), 4 (applied) |
| FR-1.7 `_test.go` tokens excluded structurally | 5 (`Tests: false`) |
| FR-2.1 generator module + `go run .` / `-check` | 5 |
| FR-2.2 `go/packages` workspace load with types | 5 |
| FR-2.3 hard failure on a partial load | 5 |
| FR-2.4/2.5 sorted `topics.yaml` with declaring packages | 5 |
| FR-2.6 conflicting policy | 5 — vacuous under design §6's single-policy-file decision; no code |
| FR-2.7 `policies.yaml`, stale entry is an error | 5 |
| FR-2.8 `-check` diffs and writes nothing | 5, 6, 7, 8 |
| FR-3.1/3.2 generated `env-configmap.yaml` block behind markers | 6 |
| FR-3.3 `kafka-topics-configmap.yaml` | 6 |
| FR-3.4 `gen-topic-config.sh` replaced by the manifest | 7 |
| FR-3.5 all three overlays regenerated | 7 |
| FR-4.1 `FromEnviron` + prefixes deleted | 11 |
| FR-4.2 mount path, configurable, fatal on unreadable | 10, 11 |
| FR-4.3 env resolution retained | 10 |
| FR-4.4 unresolvable token is fatal, named | 10 |
| FR-4.5 cleanup policy from the manifest | 10 |
| FR-4.6 all other task-260 behaviour preserved | 11 |
| FR-4.7 `Groups`/`StateIsSeedable` untouched | 11 |
| FR-5.1 `verify.sh` drift step | 8 |
| FR-5.2 trigger (analyzer as the `.go` pre-check — design §4c) | 8, 9 |
| FR-5.3 no Docker or broker | 8 |
| FR-6.1 the two `STATUS_*` tokens | 5, 6, 12 |
| FR-6.2 the 17 orphans removed | 6, 12 |
| FR-6.3 `migration.md` | 12 |
