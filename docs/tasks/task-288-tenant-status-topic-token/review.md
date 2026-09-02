# task-288 review — b2d368757

verdict: APPROVED_WITH_FINDINGS
range: `a1182780f..b2d368757` (single commit, `fix(tenants): declare the tenant status topic as a real topic.Token`)
scope_confirmed: the 14-file diff of b2d368757 — 2 PRD/findings docs, `services/atlas-tenants/.../tenant/{kafka.go,kafka_test.go,testmain_test.go}`, `tools/topicguard/{analyzer.go,testdata/.../bareliteral/x.go}`, `libs/atlas-kafka/gen/topics.yaml`, and the five regenerated deploy surfaces — plus the contracts the diff depends on: `libs/atlas-kafka/topic/{token.go,provider.go}`, `libs/atlas-kafka/producer/manager.go`, `libs/atlas-kafka/gen/policies.yaml`, `tools/gen-topics.sh`, `deploy/compose/{up.sh,docker-compose.*.yml}`, `deploy/k8s/base/atlas-tenants.yaml`, `deploy/k8s/overlays/pr-cleanup/`. The stated range matches the work found.

No blocking findings. Six non-blocking notes, two items not evaluable.

## Requirement-by-requirement

### FR-1 — retype the token

- **FR-1.1 PASS.** `services/atlas-tenants/atlas.com/tenants/tenant/kafka.go:19`:
  `const EventTopicTenantStatus topic.Token = "EVENT_TOPIC_TENANT_STATUS"`.
  Type is the named type from `libs/atlas-kafka/topic/token.go:12` (`type Token string`),
  imported at `kafka.go:8`.
- **FR-1.2 PASS.** The token is now its own `const` declaration
  (`kafka.go:19`); the residual block holds only the untyped
  `EventTypeCreated/Updated/Deleted`.
- **FR-1.3 PASS.** `processor.go:88,164,230` still read
  `mb.Put(EventTopicTenantStatus, ...)` unchanged; the argument is now
  already `topic.Token` rather than implicitly converted.
- **FR-1.4 PASS.** `testmain_test.go:13,16` sets the token to
  `"tenant-status-test"`, a value distinct from the token name, so a
  token-equals-its-own-value regression can no longer pass the suite.

### FR-2 — regenerate deploy surfaces

- **FR-2.2 PASS.** `libs/atlas-kafka/gen/topics.yaml:969-972` —
  `EVENT_TOPIC_TENANT_STATUS`, `cleanup: delete`, `packages: [atlas-tenants/tenant]`.
  Attribution format matches neighbours (`topics.yaml:964-968`).
- **FR-2.3 PASS.** `deploy/k8s/base/env-configmap.yaml:183`;
  `deploy/k8s/overlays/main/kustomization.yaml:213` (`-main`),
  `overlays/pr/kustomization.yaml:330` (`-PLACEHOLDER_ATLAS_ENV`),
  `overlays/pr-sparse/kustomization.yaml:493`
  (`-PLACEHOLDER_BASELINE_ENVIRONMENT`). Each is alphabetically placed
  between `..._TELEPORT_ROCK_STATUS`/`..._STORAGE_STATUS` and
  `..._TRADE_CUSTODY_STATUS`, consistent with the generator's ordering.
  `deploy/compose/.env.example:172` present.
  Consumption is wired: `deploy/k8s/base/atlas-tenants.yaml:26-28` does
  `envFrom: configMapRef: atlas-env`, so the new key reaches the pod
  without a per-service edit.
- **FR-2.4 PASS.** `deploy/k8s/base/kafka-topics-configmap.yaml:324-325`
  carries `cleanup: delete`; `libs/atlas-kafka/gen/policies.yaml` is
  untouched by the diff. See "cleanup policy" below for the judgement.
- **FR-2.6 PASS.** `atlas-tenants` in `deploy/compose/docker-compose.core.yml:702-719`
  declares only `LOG_LEVEL`/`DB_NAME` and inherits `env_file: .env` via
  `x-atlas-infra` (`docker-compose.core.yml:2`). No per-service topic
  enumeration to update. `docker-compose.socket.yml` does enumerate topic
  env vars per service (e.g. `:28-29`) but contains no `atlas-tenants`
  service block.
- **FR-2.1 / FR-2.5 — accepted on the author's evidence.** I did not re-run
  `gen-topics.sh --check` per the brief. The generated content is
  self-consistent with the rest of each file, which is corroborating but
  not proof that the files were generated rather than hand-edited.
  See Not evaluable.

### FR-3 — close the topicguard gap

- **FR-3.1 PASS.** `tools/topicguard/analyzer.go:200-204` — the
  `rawEnvTopicPattern` gate is gone from `reportIfUntypedConstRef`; the
  function now reports any non-`topic.Token`-typed string constant reaching
  a `topic.Token` parameter.
- **FR-3.2 PASS (widened, as the SHOULD preferred).** `analyzer.go:177-181`
  — the `*ast.BasicLit` branch retains only the `token.STRING` kind check
  and the `strconv.Unquote` error check. Rationale recorded in the package
  comment (`analyzer.go:13-22`) and in `findings.md` "Experiment".
- **FR-3.3 PASS.** `checkBareTokenLiteral` still returns early on
  `isTestFile` (`analyzer.go:115-117`); untouched by the diff.
- **FR-3.4 PASS.** `tools/topicguard/testdata/src/atlas-example/bareliteral/x.go:15,20,21`
  covers both the untyped-constant shape (`legacyTopicName = "tenant.status"`)
  and the direct-literal shape (`put("tenant.status")`), each with a `// want`
  assertion carrying the exact message.
- **FR-3.5 PASS.** `tools/topicguard/allowlist.txt` is not in the diff; no
  new entries.
- **FR-3.6 — accepted on the author's evidence.** See Not evaluable.

### FR-4 — regression coverage

- **FR-4.1 PARTIAL PASS.** `kafka_test.go:27-36` is the test that genuinely
  pins the defect: pre-fix the constant's value was `"tenant.status"`, which
  fails both `^[A-Z][A-Z0-9_]*$` (`kafka_test.go:17`) and the
  `EVENT_TOPIC_` prefix check (`kafka_test.go:33`). The two EnvProvider
  tests are weaker — see finding N2.
- **FR-4.2 PASS.** Fixture present, per FR-3.4.

## Focus-area judgements

### 1. Did anything depend on the old literal `"tenant.status"`?

No. Repo-wide grep across `*.go/*.yaml/*.yml/*.json/*.example` (excluding
`docs/`) returns only the new fixture and the new explanatory comments —
`tools/topicguard/analyzer.go:19`,
`tools/topicguard/testdata/src/atlas-example/bareliteral/x.go:15,20,21`,
`services/atlas-tenants/.../kafka_test.go:21`,
`services/atlas-tenants/.../kafka.go:14`. Zero remaining live references.
`EventTopicTenantStatus` itself is referenced only inside the `tenant`
package (`processor.go:88,164,230`, `kafka.go:19`, the two test files); no
cross-service seam. `processor_test.go` contains no `topic`/`Topic`
assertion, so no existing test pinned the old wire name — consistent with
the author's green suite. The resolution path the tests exercise is the real
one: `producer/manager.go:67` calls `topic.EnvProvider(l)(token)()`.

### 2. Is the topicguard widening sound?

Sound, with a bounded and acceptable false-positive surface.

The literal path still narrows on structure, not value: it fires only for an
argument that is *syntactically* a string `BasicLit` (or an identifier /
selector resolving to a non-`topic.Token` string constant) at a parameter
whose type is exactly `…/atlas-kafka/topic.Token`
(`analyzer.go:120-125,152-161`), in a non-`_test.go` file. Concatenations
(`BinaryExpr`), conversions, variables, and struct fields are all still out
of scope by construction.

Realistic shapes that would now be false positives and do not exist today:

- `put("")` — an empty-string sentinel token would report as
  `bare topic literal "" reaching a topic.Token parameter`, a confusing
  message for a case the guard is not really about.
- a deliberately inline throwaway token in a non-test `main.go` smoke path
  or a code generator's own fixture under `services/`.

In every such case the sanctioned remedy (declare a `topic.Token` constant)
is available and cheap, which is what makes the risk acceptable. The one
structural gap is that diagnostic 1 has **no suppression mechanism at all**:
`RawEnvAllowlist` is consulted only by diagnostic 2 (`analyzer.go:248-250`),
so a future genuinely-legitimate literal forces either a code change or an
analyzer edit. That is a deliberate trade the commit does not state; see N1.

### 3. Do the tests pin the defect?

Partly. `TestEventTopicTenantStatusIsAnEnvVarName` (`kafka_test.go:27`)
fails on the pre-fix value and is a real regression test.
`TestEventTopicTenantStatusResolvesWhenSet` (`kafka_test.go:41`) and
`TestEventTopicTenantStatusErrorsWhenUnset` (`kafka_test.go:59`) pass both
before and after the fix — pre-fix they would set/unset the variable named
`tenant.status` and `EnvProvider` behaves identically. They restate
`libs/atlas-kafka/topic/provider_test.go`'s existing `resolved` / `unset`
cases in a different package. Not harmful, but they are not the coverage
FR-4.1 asked for. See N2.

`testmain_test.go` change is correct. Package is `tenant_test`
(`testmain_test.go:1`), the same external package as `processor_test.go` and
`kafka_test.go`, so the single `TestMain` covers the whole test binary and
the token is set before any emit-path test runs.

`TestEventTopicTenantStatusErrorsWhenUnset` uses `os.Unsetenv` plus a manual
`t.Cleanup` restore rather than `t.Setenv` (which has no unset form) — correct,
and safe here because no test in `services/atlas-tenants` calls `t.Parallel`
(verified by grep across the service). If a parallel test is ever added to
this package that depends on the token, this test becomes racy; worth a
comment, not a change. See N3.

### 4. Is `cleanup: delete` right?

Yes. `libs/atlas-kafka/gen/policies.yaml:1-15` scopes `compact` to the three
configuration topics whose consumers replay from first offset at every boot
to rebuild config state. The tenant status topic has no consumer anywhere in
the repo (grep above), so no replay contract exists to protect, and `delete`
is the generator default. Adding it to `compact` would have been the defect.

### 5. Regenerated deploy surfaces

Complete and internally consistent for the three generated overlays.

- The PRD's FR-2.1 and section-10 acceptance criterion name **four** overlay
  kustomizations including `pr-cleanup`, but only three were touched. This is
  PRD imprecision, not a gap: `tools/gen-topics.sh:1-6` documents "all three
  overlay kustomizations", and `deploy/k8s/overlays/pr-cleanup/kustomization.yaml`
  contains zero `EVENT_TOPIC` lines. See N4.
- Migration/drain of the orphaned `tenant.status` topic: leaving it is the
  right call — nothing reads it, and `atlas-kafka-precreate` never created it,
  so it only exists where Kafka auto-created it on first produce. Whether it
  ages out depends on the live cluster's retention, which I cannot see. See
  Not evaluable.
- Local compose: `deploy/compose/up.sh:20-26` requires a hand-copied `.env`
  and does not merge new keys from the regenerated `.env.example`, so an
  existing local `.env` will reproduce the same 500 until re-copied. This is
  a property of every new topic, not of this change. See N5.

## Non-blocking findings

- **N1 — `tools/topicguard/analyzer.go:13-22`.** The package comment
  justifies removing the shape filter but does not state that diagnostic 1
  has no allowlist escape hatch (`RawEnvAllowlist` is diagnostic-2 only,
  `analyzer.go:248-250`). Worth one sentence, since the next engineer who
  hits a false positive will look for one.
- **N2 — `services/atlas-tenants/.../kafka_test.go:41,59`.** Both tests pass
  against the pre-fix constant and duplicate
  `libs/atlas-kafka/topic/provider_test.go`'s `resolved` / `unset` cases.
  Only `kafka_test.go:27` is load-bearing as a regression test.
- **N3 — `services/atlas-tenants/.../kafka_test.go:59-73`.** The manual
  `os.Unsetenv` + `t.Cleanup` mutates process-global state; correct today
  only because nothing in the service calls `t.Parallel`. A one-line comment
  recording that precondition would keep it correct.
- **N4 — PRD `prd.md:126-131,247`.** Names `pr-cleanup` among the overlays to
  regenerate; that overlay carries no topic literals and `gen-topics.sh`
  does not target it. The PRD text, not the commit, is wrong.
- **N5 — `deploy/compose/up.sh:20`.** Existing local `.env` files will not
  pick up `EVENT_TOPIC_TENANT_STATUS`; local `atlas-tenants` will keep 500ing
  until developers re-copy from `.env.example`. Worth a line in the PR body.
- **N6 — `services/atlas-tenants/.../processor.go:88,164,230`.** No test
  exercises the `mb.Put` emit path itself; if a future edit passed a
  different token from `processor.go`, `kafka_test.go` would not catch it.
  Mitigated by topicguard diagnostic 3 (`analyzer.go:266-298`,
  token-not-in-manifest) and the `gen-topics.sh --check` drift guard, so no
  change requested.

## Not evaluable

- **The author's verification runs.** Flagless `tools/verify.sh`,
  `./tools/gen-topics.sh --check`, the topicguard self-tests, and the
  fleet-wide `./tools/go-analyzer-guards.sh` were not re-run per the brief.
  Nothing in the reviewed code contradicts the reported results — in
  particular the generated blocks are ordering- and format-consistent with
  their neighbours, and the widened analyzer's structural narrowing makes a
  zero-violation fleet result plausible — but FR-2.1/2.5/3.6 rest on that
  evidence, not on mine.
- **Live-cluster disposition of the legacy `tenant.status` topic.** Whether
  it exists in `atlas-main` and under what retention is outside this review
  surface. The decision to abandon it is sound on repo evidence (no
  consumer); its physical cleanup is unverified.
