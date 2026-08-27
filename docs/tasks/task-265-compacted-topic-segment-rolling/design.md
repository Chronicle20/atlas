# Compacted Config-Status Topics Never Actually Compact — Design

Version: v1
Status: Draft
Created: 2026-08-26
Input: [prd.md](prd.md) (approved)

---

## 1. Scope of this document

The PRD fixes *what*: the three config-status topics must carry enough
configuration that the log cleaner actually runs on them, and the already-grown
`-main` topics must self-heal on the next sync-wave-0 Job run. This document
fixes *how*: the single shared config declaration inside
`internal/topics/topics.go`, its two projections onto the `CreateTopics` and
`IncrementalAlterConfigs` request bodies, the test surface, and — most
importantly — the resolutions of PRD §9's four open questions **against the live
`apache/kafka:4.1.1` broker**, not against remembered Kafka behaviour.

Three things came out of that live verification and changed the shape of what
ships:

- **`max.compaction.lag.ms` is the load-bearing knob, and it works by rolling
  the active segment on the next append** (§4.1). `segment.ms` at the same value
  is not "belt-and-braces" — it is arithmetically redundant, because the broker's
  effective roll trigger for a compacted topic is `min(segment.ms,
  max.compaction.lag.ms)`. It is kept, with that redundancy stated in the
  comment, for the reason given in §4.1.4.
- **The `-main` topics do self-heal** (§4.2). A segment whose first-message
  timestamp already predates the new `segment.ms`/`max.compaction.lag.ms` rolls
  on the *very next append*, and the cleaner compacts it within ~30 s. No
  remediation phase is needed. FR-3.1 stands as written.
- **The PRD's final acceptance criterion is wrong** (§4.4). `log-start-offset >
  0` is not a signal that compaction ran — compaction preserves the surviving
  records' original offsets and does not advance the log start offset. Verified
  both ways on the live broker. §4.4 gives the correct signal and §8 restates the
  criterion.

No other requirement changes. Nothing outside
`services/atlas-kafka-precreate/` is touched.

---

## 2. Live-broker verification (evidence)

All observations below are from `kafka-broker-0` in namespace `kafka`, image
`apache/kafka:4.1.1` (`kafka-topics.sh --version` → `4.1.1`), on 2026-08-26
between 15:07 and 15:11 UTC. Two scratch topics were created for the experiment
and **deleted afterwards** (`kafka-topics.sh --list | grep TASK265` → no
matches).

### 2.1 The defect, reproduced from the broker's own state

```
$ kafka-configs.sh --entity-type topics \
    --entity-name EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main --describe
Dynamic configs for topic EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main are:
  cleanup.policy=compact sensitive=false synonyms={DYNAMIC_TOPIC_CONFIG:cleanup.policy=compact, DEFAULT_CONFIG:log.cleanup.policy=delete}

$ kafka-get-offsets.sh --topic EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main --time -2
EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main:0:0
$ kafka-get-offsets.sh --topic EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main --time -1
EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main:0:36095

$ ls -la /var/lib/kafka/data/kafka/EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main-0
-rw-rw-r-- 1 appuser root 10485760 Aug 26 15:04 00000000000000000000.index
-rw-rw-r-- 1 appuser root  9805952 Aug 26 15:05 00000000000000000000.log
-rw-rw-r-- 1 appuser root 10485756 Aug 26 15:04 00000000000000000000.timeindex
```

One segment, base offset 0, never rolled, `cleanup.policy=compact` and nothing
else. Exactly the PRD's §1 claim.

The cleaner itself is healthy and *is* running on this broker — the
cleaner-offset checkpoint has 52 entries, including two of the three config
topics:

```
$ grep -i CONFIGURATION /var/lib/kafka/data/kafka/cleaner-offset-checkpoint
EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS-main 0 133
EVENT_TOPIC_CONFIGURATION_TENANT_STATUS-main 0 134
```

`ENVIRONMENT_STATUS` is absent. This is the sharpest available statement of the
bug: the cleaner has nothing to do on that topic because there is no rolled
(non-active) segment to clean, while the two sibling topics — whose lower,
burstier write pattern happened to roll a segment at some point — have been
cleaned. Their directories show the rolled/cleaned structure the environment
topic lacks (`SERVICE_STATUS-main-0`: a 12 KB `…18.log` plus the active
`…133.log`; log-start-offset 18, end offset 141).

### 2.2 Experiment A — `max.compaction.lag.ms` alone

Topic `TASK265-SCRATCH-A`, created with `cleanup.policy=compact`,
`max.compaction.lag.ms=60000`, `min.cleanable.dirty.ratio=0.01`, and **no
`segment.ms`**. 200 records over 3 keys were produced at 15:07.

At 15:10:41 — 3 min 40 s later, well past the 60 s lag — nothing had happened:

```
-rw-r--r-- 1 appuser root 2689 Aug 26 15:07 00000000000000000000.log
TASK265-SCRATCH-A:0:0          (log start offset)
(no TASK265 entry in cleaner-offset-checkpoint)
```

A **single record** was then appended. Immediately:

```
-rw-r--r-- 1 appuser root 2689 Aug 26 15:07 00000000000000000000.log
-rw-r--r-- 1 appuser root   77 Aug 26 15:10 00000000000000000200.log
```

and within the next cleaner pass:

```
$ grep TASK265 …/cleaner-offset-checkpoint
TASK265-SCRATCH-A 0 200

$ ls -la …/TASK265-SCRATCH-A-0
-rw-r--r-- 1 appuser root      103 Aug 26 15:07 00000000000000000000.log
-rw-r--r-- 1 appuser root     2689 Aug 26 15:07 00000000000000000000.log.deleted
-rw-r--r-- 1 appuser root       77 Aug 26 15:10 00000000000000000200.log
```

2689 bytes → 103 bytes; 200 records → 3 surviving keys.

**Conclusion (OQ-1):** `max.compaction.lag.ms` *is* the mechanism, and it works
by lowering the segment-roll threshold, evaluated **on append**. It does not
force a roll on a quiescent topic. `segment.ms` was not needed for any of it.

### 2.3 Experiment B — self-healing an already-old segment

Topic `TASK265-SCRATCH-B`, created with only `cleanup.policy=compact` and loaded
with the same 200 records at 15:07–15:08. At 15:09:28 the three new configs were
applied with the same `kafka-configs.sh --alter --add-config` shape the tool's
`IncrementalAlterConfigs` produces — i.e. the topic's active segment already had
a first-message timestamp older than the new 60 s `segment.ms`, which is exactly
the `-main` situation FR-3.2 asks about. Then **one** record was appended:

```
-rw-r--r-- 1 appuser root        0 Aug 26 15:09 00000000000000000000.index
-rw-r--r-- 1 appuser root     2689 Aug 26 15:08 00000000000000000000.log
-rw-r--r-- 1 appuser root 10485760 Aug 26 15:09 00000000000000000200.index
-rw-r--r-- 1 appuser root       77 Aug 26 15:09 00000000000000000200.log
```

The old segment rolled on that first append — no waiting period, no accumulation
threshold. ~30 s later the cleaner had processed it
(`TASK265-SCRATCH-B 0 200` in the checkpoint, `…0000.log.deleted` present), and
a full `--from-beginning` consume returned:

```
Offset:197  k0  v198
Offset:198  k1  v199
Offset:199  k2  v200
Offset:200  k0  trigger
```

4 records retained out of 201.

**Conclusion (OQ-2):** the existing `-main` topics self-heal. The environment
topic takes an append every 30 s from `StartHeartbeat`, so the roll fires within
30 s of the Job's alter landing, and the cleaner collapses the 9.8 MB segment on
its next pass. FR-3.1 needs no remediation step.

### 2.4 Index preallocation (OQ-3)

The 10 MB `.index`/`.timeindex` files are preallocated for the **active** segment
only; on roll they are trimmed to their actual content. Both experiments show it
directly — after rolling, the previously-active segment's index went to
`0` bytes (A and B) and its timeindex to `12` bytes, while the new active
segment got fresh 10485760/10485756-byte files. The `-main` siblings show the
same steady state (`…18.index` = 16 bytes, `…29.index` = 72 bytes, active
`…133.index` = 10485760 bytes).

**Conclusion (OQ-3):** 144 rolls/day does **not** cost 144 × 20 MB. The
steady-state footprint per compacted partition is one active segment's
preallocated pair (~20 MB of sparse file) plus the compacted tail, which for
these topics is kilobytes. `segment.index.bytes` does **not** need lowering, and
the design does not touch it. NFR-2 is satisfied.

### 2.5 OQ-4 — Job cadence

Out of scope for code, as the PRD says. Recorded here for the operator: the
`atlas-main` namespace's `atlas-kafka-precreate` Job had completed 7 m 45 s
before the observation window, so each ArgoCD sync of that namespace re-runs it;
the `-main` topics converge on the first sync after this change lands, and the
sparse PR envs — which read the baseline topic — benefit from that one run
without needing their own.

---

## 3. Design

### 3.1 The shared declaration

The defect class this task fixes is a config set that exists in one request
builder and not the other. The design makes that state unrepresentable: one
package-level slice, two pure projections, no literal config name or value at
either call site.

```go
// internal/topics/topics.go

const (
	compactCleanupPolicy = "compact"

	// compactMaxCompactionLagMs bounds how long a record may sit
	// uncompacted. This is the knob that makes cleanup.policy=compact
	// mean anything: the cleaner never touches a partition's ACTIVE
	// segment, so a topic whose segment never rolls is never cleaned no
	// matter what its cleanup policy says. For a compacted topic the
	// broker's effective roll deadline is min(segment.ms,
	// max.compaction.lag.ms), so setting this to 10 minutes makes the
	// segment roll ten minutes after its first record — and the roll is
	// what hands the cleaner something to work on.
	compactMaxCompactionLagMs = "600000"

	// compactSegmentMs is the same 10-minute deadline expressed on the
	// policy-independent knob. It is deliberately equal to
	// compactMaxCompactionLagMs and is therefore redundant while the
	// topic is compacted (the broker takes the min of the two). It is
	// set anyway so the roll bound survives someone changing
	// cleanup.policy, and so the roll cadence is legible from a
	// --describe without knowing the min() rule.
	compactSegmentMs = "600000"

	// compactMinCleanableDirtyRatio lets the cleaner select a segment
	// whose dirty fraction is small, rather than the 0.5 default.
	compactMinCleanableDirtyRatio = "0.01"
)

// compactTopicConfig is one (name, value) pair in the configuration every
// compacted topic must carry. It exists so the CreateTopics and
// IncrementalAlterConfigs request bodies are two projections of one
// declaration: a config applied at creation but not at alter (or the
// reverse) is precisely the defect this set was added to fix.
type compactTopicConfig struct {
	name  string
	value string
}

var compactTopicConfigs = []compactTopicConfig{
	{name: "cleanup.policy", value: compactCleanupPolicy},
	{name: "max.compaction.lag.ms", value: compactMaxCompactionLagMs},
	{name: "segment.ms", value: compactSegmentMs},
	{name: "min.cleanable.dirty.ratio", value: compactMinCleanableDirtyRatio},
}

func compactCreateEntries() []kafka.ConfigEntry {
	entries := make([]kafka.ConfigEntry, len(compactTopicConfigs))
	for i, cfg := range compactTopicConfigs {
		entries[i] = kafka.ConfigEntry{ConfigName: cfg.name, ConfigValue: cfg.value}
	}
	return entries
}

func compactAlterConfigs() []kafka.IncrementalAlterConfigsRequestConfig {
	configs := make([]kafka.IncrementalAlterConfigsRequestConfig, len(compactTopicConfigs))
	for i, cfg := range compactTopicConfigs {
		configs[i] = kafka.IncrementalAlterConfigsRequestConfig{
			Name:            cfg.name,
			Value:           cfg.value,
			ConfigOperation: kafka.ConfigOperationSet,
		}
	}
	return configs
}
```

Both projections build a fresh slice per call. `kafka.TopicConfig.ConfigEntries`
and `IncrementalAlterConfigsRequestResource.Configs` are per-topic fields that
kafka-go reads but does not document as immutable; sharing one backing array
across N resources would be a silent aliasing hazard for no measurable gain at
these sizes (3 topics × 4 configs).

Why a local `compactTopicConfig` pair type rather than declaring the canonical
set as `[]kafka.ConfigEntry` and projecting only one way: the alter direction
needs a third field (`ConfigOperation`) that has no counterpart in
`kafka.ConfigEntry`, so one of the two directions is a projection either way. A
neutral pair type makes the declaration read as *policy*, keeps `kafka` types out
of it, and makes the "these two request bodies agree" test assert against the
same source both sides derive from.

### 3.2 Call-site changes in `Ensure`

Two edits, both replacing an inline literal with a call:

```go
for _, name := range t.Compact {
	cfgs = append(cfgs, kafka.TopicConfig{
		Topic:             name,
		NumPartitions:     1,
		ReplicationFactor: 1,
		ConfigEntries:     compactCreateEntries(),
	})
}
```

```go
resources[i] = kafka.IncrementalAlterConfigsRequestResource{
	ResourceType: kafka.ResourceTypeTopic,
	ResourceName: name,
	Configs:      compactAlterConfigs(),
}
```

Everything else in `Ensure` is unchanged, and each unchanged part is load-bearing
for a requirement:

- The plain-topic loop still emits no `ConfigEntries` (FR-1.7).
- The alter still runs unconditionally over all of `t.Compact`, not over the
  subset `CreateTopics` reported as newly created — this is the entire
  self-healing mechanism (FR-1.5, FR-3.1), now confirmed by §2.3.
- Still one `IncrementalAlterConfigs` request, one resource per compacted topic;
  the resource count does not change, only each resource's `Configs` length
  (FR-1.6).
- Still `ConfigOperationSet` on the incremental API, never the legacy
  full-replace `AlterConfigs`. The existing comment block explaining why keeps
  its rationale and gains the §2 explanation of why `cleanup.policy` alone was
  inert (FR-1.4, FR-4.2).
- The `len(t.Compact) == 0` short circuit before the alter is unchanged.

### 3.3 Error paths

`Ensure`'s two alter-related error wraps currently read `"setting
cleanup.policy=compact"`. With four configs that name is now a lie about what
failed, so both become `"applying compacted topic config"` /
`"applying compacted topic config on topic %q"`. The **behaviour** does not
change: transport error → fatal; any per-resource error → fatal, joined, naming
the topic (FR-2.1).

This is also FR-2.2 in full. There is no allowlist of "configs we tolerate the
broker rejecting" and no partial-application fallback: if a broker does not know
`max.compaction.lag.ms`, `IncrementalAlterConfigs` returns
`INVALID_CONFIG`/`UNKNOWN_TOPIC_OR_PARTITION` on that resource, `Ensure` returns
it, and the Job exits 1. A silently-uncompacted compacted topic is the bug being
fixed; it must not be reachable through a swallowed error. (Not a live risk on
this cluster — `max.compaction.lag.ms` is KIP-354, Kafka 2.3+, and the broker is
4.1.1 with `DEFAULT_CONFIG:log.cleaner.max.compaction.lag.ms=9223372036854775807`
visible in §2.3's `--describe`.)

FR-2.3 (re-applying an identical value is a no-op) needs no code: `--alter
--add-config` with unchanged values returned `Completed updating config` in §2.3
and the tool's own runs have been re-applying `cleanup.policy=compact` on every
Job run since task-260.

### 3.4 Logging (NFR-5)

`main.go`'s alter log line currently carries `{"phase":"alter","topics":N,
"policy":"compact"}`. It gains one field and loses none:

```go
logrus.WithFields(logrus.Fields{
	"phase":   "alter",
	"topics":  len(t.Compact),
	"policy":  "compact",
	"configs": topics.CompactConfigNames(),
}).Info("compacted topic configuration applied")
```

`topics.CompactConfigNames() []string` is the one exported accessor this design
adds — names only, derived from `compactTopicConfigs`, so the log cannot drift
from what was sent. Values are omitted: they are four short numbers that would
double the line's width and are recoverable from `--describe`. The message text
changes from "cleanup policy applied" to match what now happens.

### 3.5 What is deliberately not done

- **No new `discover` category.** `Topics.Plain` / `Topics.Compact` already
  carries the classification (PRD §6); `compactVars` stays at three entries.
- **No `segment.index.bytes` override.** §2.4 shows the preallocation cost does
  not scale with roll count.
- **No `retention.ms` / `delete.retention.ms` change.** Compaction, not
  deletion, is the mechanism; tombstone retention is not implicated.
- **No remediation phase, flag, or one-shot Job.** §2.3 proved the ordinary
  alter path heals the existing topics.
- **No deploy/overlay change.** Confirmed by the shape of the change: every knob
  is applied programmatically from `topics.go`.
- **No change to the heartbeat cadence, the projection replay model, or the
  plain topic set.** All three are PRD §2 non-goals.

---

## 4. Open-question resolutions

| OQ | Resolution | Evidence |
| --- | --- | --- |
| **OQ-1** — does `max.compaction.lag.ms` force-roll on 4.1.1? | **Yes, on append.** The compacted-topic roll deadline is `min(segment.ms, max.compaction.lag.ms)`; a topic with only `max.compaction.lag.ms=60000` and no `segment.ms` rolled and compacted (2689 B → 103 B, 200 records → 3). It does **not** roll a quiescent topic: A sat 3 m 40 s past its lag with no roll until one record arrived. `segment.ms` at the same value is redundant, and is kept for the reasons in §3.1. | §2.2 |
| **OQ-2** — does the existing 9.8 MB segment self-heal? | **Yes, on the next append.** A segment whose first-message timestamp already predates the newly-applied `segment.ms` rolled on the very next record, with no accumulation threshold; the cleaner collapsed it ~30 s later. With a 30 s heartbeat, `-main` converges within ~30 s of the Job's alter. | §2.3 |
| **OQ-3** — index preallocation cost of 144 rolls/day? | **Not a cost.** Index/timeindex preallocation applies to the active segment only and is trimmed to actual size on roll (10485760 B → 0 B observed). No `segment.index.bytes` change. NFR-2 satisfied. | §2.4 |
| **OQ-4** — when does each namespace converge? | ArgoCD sync cadence; each sync re-runs the wave-0 Job. `atlas-main`'s run is what repairs the topic sparse PR envs read. No code implication. | §2.5 |

### 4.1 Why `segment.ms` and `min.cleanable.dirty.ratio` are still shipped

Being honest about what the evidence does and does not show, since FR-4.2
requires a future reader be able to tell:

1. `max.compaction.lag.ms=600000` is sufficient on its own for the observed
   workload. §2.2 is a clean isolation of it.
2. `segment.ms=600000` is exactly redundant *given* the min() rule and *given*
   `cleanup.policy=compact`. It is retained because it is the knob that keeps
   holding if the cleanup policy is ever changed, and because it makes the roll
   cadence readable from `--describe` without knowing the min() rule.
3. `min.cleanable.dirty.ratio=0.01` was **not** isolated by either experiment —
   both ran at a dirty ratio near 1.0, far above the 0.5 default, so neither
   shows it doing work. It is retained as cheap insurance for the steady state
   (a few clean records plus one 10-minute window of heartbeats), and because
   the forced-cleaning path that `max.compaction.lag.ms` drives bypasses the
   ratio check anyway, making it inert-but-harmless rather than load-bearing.

The in-code comments say this in the same terms. A future reader must not be
able to conclude the extra knobs are load-bearing when they are not, nor that
they are free to delete `max.compaction.lag.ms` because `segment.ms` "covers
it" — the comment on `compactSegmentMs` names which one is doing the work.

### 4.2 Correction to the PRD's final acceptance criterion

PRD §10's last bullet asks for `kafka-get-offsets.sh --time -2` to report a
log-start-offset **greater than 0** as "the direct signal that compaction has run
at least once." **That signal is wrong**, and both experiments show it:
`TASK265-SCRATCH-A` and `-B` were each demonstrably compacted (201 records → 4,
old segment `.deleted`, cleaner checkpoint at offset 200) and both still report
log-start-offset `0`.

Compaction rewrites a segment in place, keeping surviving records at their
original offsets; it does not advance `logStartOffset`. The log start offset only
moves when an entire head segment is dropped — which is why the two sibling
topics show 18 and 29 (their earliest segment eventually emptied), and why
`ENVIRONMENT_STATUS-main`, with everything in one base-0 segment, will keep
reporting `0` even after it is fully compacted.

The correct post-deploy signals, in decreasing sharpness:

1. `grep EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main
   /var/lib/kafka/data/kafka/cleaner-offset-checkpoint` returns a line. Today it
   returns nothing (§2.1). This is the direct statement "the cleaner has
   processed this partition."
2. `ls -la …/EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main-0` shows more than
   one `.log` and a base-0 `.log` far smaller than 9.8 MB.
3. A `--from-beginning` consume returns on the order of tens of records rather
   than 36 000.

§8 restates the criterion accordingly. Everything else in PRD §10 stands.

---

## 5. Test surface

All in `internal/topics/topics_test.go`, against the existing `stubClient`
(NFR-6 — no broker required).

1. **`TestEnsure_CompactConfigsMatchAcrossRequests`** — the anti-regression test
   for the defect class. Runs `Ensure` with one plain and two compacted topics,
   then asserts that the set of `(name, value)` pairs on every compacted
   `kafka.TopicConfig.ConfigEntries` equals the set on every
   `IncrementalAlterConfigsRequestResource.Configs`, and that both equal the
   expected four pairs. Written as a set comparison, so it does not encode
   slice order.
2. **`TestEnsure_CreateTopics_CompactConfigEntries`** (extends the existing
   create assertions) — table-driven over the exact four
   `kafka.ConfigEntry{ConfigName, ConfigValue}` values from FR-1.1, including
   the literal strings `"600000"`, `"600000"`, `"0.01"`. Asserting the literal
   values, not the constants, is deliberate: a test that reads the constant
   cannot catch someone changing the constant.
3. **`TestEnsure_AlterConfigs`** (existing, extended) — same four values as
   `IncrementalAlterConfigsRequestConfig`, each with
   `ConfigOperation: kafka.ConfigOperationSet`; still exactly one alter call and
   one resource per compacted topic; `len(res.Configs)` goes 1 → 4.
4. **`TestEnsure_PlainTopicsCarryNoConfig`** (the negative case, FR-1.7) — a
   plain topic's `ConfigEntries` is empty *and* the plain topic name appears in
   no alter resource. The current suite checks the second half inside the
   compacted loop, where it can never fire; this makes it a standalone
   assertion.
5. **`TestEnsure_AlterConfigs_ResourceError`** (existing) — unchanged behaviour,
   updated only for the new error-message wording; still asserts the failing
   topic name appears in the error.
6. **`TestEnsure_AlterConfigs_NoCompactTopics`** (existing) — unchanged.

`main.go`'s logging change is covered by the existing exercise of that path; a
tiny `TestCompactConfigNames` asserts the exported accessor returns the four
names in declaration order.

---

## 6. Documentation (FR-4.1, FR-4.2)

`services/atlas-kafka-precreate/README.md`:

- The `COMMAND_TOPIC_*` / `EVENT_TOPIC_*` env-var row stops saying "created with
  `cleanup.policy=compact`" and names all four configs.
- The **Create/Alter** phase bullet gains a sentence stating that
  `cleanup.policy` alone is inert — the cleaner never touches a partition's
  active segment — and that `max.compaction.lag.ms` is what makes the segment
  roll and therefore what makes the policy mean anything.
- A short "Compacted topics" subsection carries the four-row table from PRD
  §4.1 plus the §4.1-above honesty about which knob is load-bearing.

In-code: the comment block quoted in §3.1, plus the extension to the existing
`IncrementalAlterConfigs` rationale comment described in §3.2.

---

## 7. Risks

| Risk | Assessment |
| --- | --- |
| A broker in some environment predates KIP-354 (Kafka < 2.3) and rejects `max.compaction.lag.ms`. | The Job fails loudly and that environment's wave-0 blocks. This is FR-2.2's intended behaviour, not a regression. The only broker in play is 4.1.1 (§2.1). |
| 10-minute rolls churn segments on a topic that is quiet. | A quiet topic does not roll at all — the roll is append-triggered (§2.2). Churn is bounded by write rate, and at the 30 s heartbeat that is ≤ 144 rolls/day/topic, whose cost §2.4 shows is the actual bytes, not the preallocated ones. |
| Compaction loses a config record a projection needs. | Compaction retains the latest record per key, which is exactly the projection's model (`discover.go`'s `compactVars` comment). Tombstones are not emitted by the outbox for these topics. |
| `-main` does not converge because its Job does not re-run. | OQ-4. Operator-visible, not a code risk; §2.5 records that the Job had run 7 m before the observation, so the namespace does re-run it. |

---

## 8. Acceptance criteria (as amended)

Unchanged from PRD §10 except the last bullet:

- [ ] `topics.go` declares `max.compaction.lag.ms`, `segment.ms`, and
      `min.cleanable.dirty.ratio` as named constants with the FR-1.1 values.
- [ ] Both request builders derive from `compactTopicConfigs` (FR-1.3), and a
      test asserts the two request bodies agree.
- [ ] Alter path still `ConfigOperationSet`, still unconditional over the whole
      compacted set (FR-1.4, FR-1.5), still one request (FR-1.6).
- [ ] Plain topics carry no `ConfigEntries` and appear in no alter resource,
      asserted standalone (FR-1.7).
- [ ] A per-resource alter error remains fatal and names the topic (FR-2.1,
      FR-2.2).
- [ ] `go build ./...` and `go test ./...` pass in
      `services/atlas-kafka-precreate`; `tools/verify.sh` (flagless) exits 0.
- [ ] README and in-code comments state the full configuration and why
      `cleanup.policy` alone is inert (FR-4.1, FR-4.2).
- [ ] **Amended (§4.2):** post-deploy on the live cluster, after the
      `atlas-main` wave-0 Job re-runs,
      `kafka-configs.sh --entity-type topics --entity-name EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main --describe`
      shows all four configs, **and**
      `grep EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS-main /var/lib/kafka/data/kafka/cleaner-offset-checkpoint`
      returns a line (it returns nothing today), **and** the partition directory
      shows a base-0 `.log` far below 9.8 MB. Log-start-offset is *not* the
      signal — see §4.2.
