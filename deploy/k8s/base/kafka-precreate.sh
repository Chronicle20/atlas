#!/usr/bin/env bash
# deploy/k8s/base/kafka-precreate.sh
#
# Body of the sync-wave-0 atlas-kafka-precreate Job (atlas-kafka-precreate.yaml).
# Extracted to its own file (task-232 Task 45) so atlas-kafka-precreate_test.sh
# can source it and exercise seed_group directly — a script that exists only
# inside a YAML heredoc cannot be unit tested.
#
# Mounted into the Job via the atlas-kafka-precreate-script configMapGenerator
# entry in kustomization.yaml and invoked as `bash kafka-precreate.sh`. When
# sourced (`. kafka-precreate.sh`) instead of executed, main is NOT run — the
# `(return 0 2>/dev/null)` probe below distinguishes the two — so a test can
# load the function definitions without BOOTSTRAP_SERVERS or a live Kafka.
set -euo pipefail

# apache/kafka:3.7.2's PATH does not include the Kafka CLI tools — only
# /opt/java/openjdk/bin and the usual system dirs (confirmed empirically,
# task-232 Task 45 review round 1: `docker run --rm apache/kafka:3.7.2 sh -c
# "command -v kafka-consumer-groups.sh"` exits 127). precreate_topics below
# already uses the full /opt/kafka/bin/ path throughout; seed_group and
# verify_group_offsets must too, or the first overlay that ever sets
# KAFKA_CONSUMER_GROUP turns exit 127 into a Job crash-loop under
# set -euo pipefail — and every Deployment sits at sync-wave 10 behind it.
KAFKA_BIN="/opt/kafka/bin"

# precreate_topics diffs the desired topic list against the broker's with comm,
# which requires both sides sorted under the SAME collation. Topic names mix `_`
# and `-` (COMMAND_TOPIC_ACCOUNT_LOGOUT-main), exactly where locale collation
# and byte order disagree — a mismatch makes comm emit spurious "missing"
# entries and silently skip real ones. Pin every sort and comm here to byte
# order so the diff can never depend on the image's locale.
export LC_ALL=C

# Pre-create every Kafka topic this env will use, before any service consumer
# starts. Atlas services rely on Kafka topic auto-create on first publish, and
# many subscribe to the same topic concurrently at startup — the result is a
# "consumer fetch wedged: exceeded consecutive timeouts" stampede where
# kafka-go's Readers wedge and rebalance until topics materialise.
#
# Reads the Job's own envFrom-injected atlas-env ConfigMap, picks out every
# COMMAND_TOPIC_* / EVENT_TOPIC_* value, and calls kafka-topics.sh
# --create --if-not-exists once per topic.
#
# Idempotent: re-running re-creates nothing; existing topics are left alone
# (--if-not-exists). Sets the global $topics / $compact_topics / $count
# variables (bash functions are not variable-scoped without `local`), reused
# by seed_override_offsets and verify_group_offsets below.
precreate_topics() {
    : "${BOOTSTRAP_SERVERS:?BOOTSTRAP_SERVERS not set in atlas-env}"
    echo "[$(date -u +%FT%TZ)] precreating topics on $BOOTSTRAP_SERVERS"

    # Collect the distinct topic names from every COMMAND/EVENT_TOPIC_* env var.
    # Config-projection topics go in a separate list: their consumers
    # replay them from first-offset at every boot to rebuild
    # tenant/service config state, and the outbox never re-emits a
    # (topic, key) it already delivered. With the default DELETE
    # cleanup, retention empties the topic ~7 days after the last
    # config change and every later projection boot has nothing to
    # replay (2026-07-08 world/channel/character-factory loops).
    # Events are keyed (tenant:<uuid> / service:<uuid>), so
    # compaction retains the latest snapshot per key forever.
    topics="$(mktemp)"
    compact_topics="$(mktemp)"
    for var in $(compgen -e | grep -E '^(COMMAND|EVENT)_TOPIC_' || true); do
        topic="${!var}"
        if [ -z "$topic" ]; then
            continue
        fi
        case "$var" in
            EVENT_TOPIC_CONFIGURATION_TENANT_STATUS|EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS|EVENT_TOPIC_CONFIGURATION_ENVIRONMENT_STATUS)
                printf '%s\n' "$topic" >> "$compact_topics"
                ;;
            *)
                printf '%s\n' "$topic" >> "$topics"
                ;;
        esac
    done
    sort -u "$compact_topics" -o "$compact_topics"
    # A topic named by both a config-status var and any other var must
    # stay compacted — drop it from the plain list.
    sort -u "$topics" | comm -23 - "$compact_topics" > "${topics}.plain"
    mv "${topics}.plain" "$topics"
    count=$(( $(wc -l < "$topics") + $(wc -l < "$compact_topics") ))

    # Every kafka-topics.sh call is a full JVM cold start (~1-1.5s of almost
    # pure CPU: classload, JIT warmup, admin-client bootstrap, one metadata
    # fetch) and creates exactly ONE topic — the CreateTopics RPC itself is a
    # rounding error inside that. Blindly running --create --if-not-exists once
    # per topic therefore costs ~170 JVM starts on EVERY sync, even though on a
    # re-sync all 170 already exist and every one of those JVMs does nothing but
    # confirm it. Three environments syncing together pegged an 8-core node at
    # 99% for minutes at a time (2026-08-21, eos).
    #
    # So: ask the broker ONCE which topics exist (one JVM), and only create the
    # difference. Steady-state re-sync collapses from ~170 JVM starts to 1.
    existing="$(mktemp)"
    /opt/kafka/bin/kafka-topics.sh \
        --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --list \
        | sort -u > "$existing"

    # comm needs both sides sorted; $topics and $compact_topics already are.
    missing="$(mktemp)"
    missing_compact="$(mktemp)"
    comm -23 "$topics" "$existing" > "$missing"
    comm -23 "$compact_topics" "$existing" > "$missing_compact"
    missing_count=$(( $(wc -l < "$missing") + $(wc -l < "$missing_compact") ))

    # Fan out the remainder so the JVM starts overlap, but at -P 4 rather than
    # -P 16: the Job is one of several syncing concurrently on a shared node and
    # 16 JVM cold starts saturate a whole node's CPU on their own. With the
    # existing-topic filter above, the fan-out now only ever runs on a genuinely
    # new set, where a lower width costs seconds, not minutes. --if-not-exists
    # is kept as a race guard (another env's Job may create the same topic
    # between our --list and our --create). xargs exits non-zero if any child
    # fails, which set -e turns into a Job retry.
    echo "[$(date -u +%FT%TZ)] $count topics desired, $missing_count missing — creating (concurrency 4)"
    xargs -P 4 -I {} \
        /opt/kafka/bin/kafka-topics.sh \
            --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --create --if-not-exists \
            --topic {} \
            --partitions 1 \
            --replication-factor 1 \
        < "$missing" >/dev/null

    # The compacted config-projection topics (only a handful) are
    # created serially with cleanup.policy=compact.
    while read -r topic; do
        /opt/kafka/bin/kafka-topics.sh \
            --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --create --if-not-exists \
            --topic "$topic" \
            --partitions 1 \
            --replication-factor 1 \
            --config cleanup.policy=compact \
            >/dev/null
    done < "$missing_compact"

    # Topics created before the compact policy existed still need converging,
    # and those are by definition ones that ALREADY exist — so this alter pass
    # runs over $compact_topics, not $missing_compact. It is a handful of
    # topics, and --add-config is idempotent.
    while read -r topic; do
        /opt/kafka/bin/kafka-configs.sh \
            --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --alter --entity-type topics --entity-name "$topic" \
            --add-config cleanup.policy=compact \
            >/dev/null
    done < "$compact_topics"

    # Name the created count explicitly: before the existing-topic diff above,
    # this line meant "issued a create for all $count". It now means "$count
    # desired, of which $missing_count were actually created", and a reader
    # tracking down a missing topic needs to see which.
    echo "[$(date -u +%FT%TZ)] reconciled $count topics ($missing_count created, $(( count - missing_count )) already present)"
}

# Probe one consumer group's state. Echoes the state token (Empty, Dead,
# Stable, PreparingRebalance, CompletingRebalance, …) or the empty string.
#
# Parsing, per the measured apache/kafka:3.7.2 output (design §2.1): the
# command prints a BLANK first line, then a header, then one data row. Real
# group names contain spaces and brackets (e.g. "Channel Service - <uuid>
# [f8c5]"), which shift every fixed forward column — but a row always ends in
# exactly five whitespace-separated tokens (COORDINATOR (ID)
# ASSIGNMENT-STRATEGY STATE #MEMBERS), so STATE is $(NF-1) regardless of how
# many tokens GROUP contributes. This is the same NF-anchored idiom
# verify_group_offsets already documents (FR-1.5).
#
# The header MUST be excluded by $(NF-1)!="STATE", not by line number and not
# by field count: a single-token group name yields NF=6, exactly the header's
# NF, so a count-based guard would "find" a state of "STATE".
#
# STATE is parsed rather than #MEMBERS (OQ-2) because FR-4.1 requires the log
# line to name the state anyway, a string allowlist rejects garbage without a
# separate numeric guard, and #MEMBERS collapses Dead, Empty and any future
# zero-member-but-active state into one bucket.
#
# A nonexistent group exits 0 with no data row and an Error: line on stdout
# (design §2.2), so absence needs no special case — it yields "". Any
# non-zero exit and any unparseable output collapse to "" as well, which
# state_is_seedable calls seedable (FR-1.4): the probe can never itself fail
# the Job.
group_state() {
    set +e
    state_out="$("$KAFKA_BIN/kafka-consumer-groups.sh" \
        --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --group "$1" --describe --state 2>/dev/null)"
    set -e
    printf '%s\n' "$state_out" | awk 'NF>=6 && $(NF-1)!="STATE" { print $(NF-1); exit }'
}

# Classify a consumer-group state token as seedable (offsets may be reset) or
# active (a live member holds the group; Kafka will refuse the reset).
#
# Deliberately an ALLOWLIST: Empty, Dead and "" are seedable, everything else
# is active. "" covers an absent group, unparseable --describe --state output,
# and a failed probe (FR-1.4) — all of which must fall through to seed_group,
# whose own message classifier then governs. A state token Kafka adds in a
# future version falls into "active" and is skipped, which is the safe
# direction: skipping never mutates a committed offset (FR-5.2), whereas a
# denylist would reset a group in an unrecognised live state.
#
# Pure — no Kafka call, no I/O — so atlas-kafka-precreate_test.sh asserts the
# whole truth table without a broker (design §3.1).
state_is_seedable() {
    case "$1" in
        Empty|Dead|"") return 0 ;;
        *) return 1 ;;
    esac
}

# Commit end-of-log offsets for an override consumer group on one or more
# topics in a SINGLE kafka-consumer-groups.sh invocation. On a FIRST sync the
# group is empty and therefore resettable; on a RE-SYNC of a live environment
# the Job is deleted and recreated (Force=true,Replace=true) while the
# environment's Deployments have been joined to those very groups for hours,
# so the group is active and Kafka refuses the reset. seed_override_offsets
# probes for that ahead of time (group_state / state_is_seedable); this
# function classifies the race that remains between the probe and the reset.
#
# MEASURED (design §2.3, apache/kafka:3.7.2, both --dry-run and --execute,
# both single- and repeated---topic form): a reset refused because the group
# is active EXITS 0 and prints
#   Error: Assignments can only be reset if the group '<name>' is inactive,
#   but the current state is <State>.
# to STDOUT. Classification therefore keys off the MESSAGE, and the message
# case must run BEFORE the exit-code check — a code-only classifier is
# decorative. Before this change that output went to /dev/null and the
# refusal was a silent no-op reporting success.
#
# Returns: 0 seeded · 2 refused-because-active (non-fatal, FR-2.2) ·
# anything else fatal (FR-2.3 — broker-unreachable, authorization and
# malformed-argument failures still fail the Job). Callers MUST use
# `rc=0; seed_group … || rc=$?` so set -e does not abort on a deliberate 2.
#
# The matched substring deliberately stops before the group name and the
# state, so it survives a changed quoting style, a new state name, and a
# group name containing glob metacharacters.
#
# Takes a group name followed by one-or-more topic names and repeats --topic
# once per topic: empirically confirmed against apache/kafka:3.7.2 (task-232
# Task 45 review round 1) that --reset-offsets --execute accepts repeated
# --topic and resets every named topic in that one call, not just the last —
# see atlas-kafka-precreate_test.sh's multi-topic assertion. This collapses
# what would otherwise be O(groups × topics) JVM cold starts (~1-1.5s each,
# per precreate_topics' comment above) down to O(groups).
#
# Topic names are plain identifiers (no spaces/glob metacharacters), so the
# unquoted expansion below is intentional word-splitting into repeated
# --topic flags, not a bug.
seed_group() {
    group="$1"; shift
    topic_args=""
    for topic in "$@"; do
        topic_args="$topic_args --topic $topic"
    done

    # set +e spans exactly one command substitution and its $? read — the
    # minimum region the failure-isolation NFR allows. 2>&1 capture replaces
    # the old >/dev/null (FR-2.4); the success path stays silent because the
    # capture is only printed on the fatal branch.
    set +e
    # shellcheck disable=SC2086
    seed_out="$("$KAFKA_BIN/kafka-consumer-groups.sh" --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --group "$group" $topic_args --reset-offsets --to-latest --execute 2>&1)"
    seed_rc=$?
    set -e

    case "$seed_out" in
        *"Assignments can only be reset if the group"*) return 2 ;;
    esac
    if [ "$seed_rc" -ne 0 ]; then
        printf '%s\n' "$seed_out" >&2
        return "$seed_rc"
    fi
    return 0
}

# Seeds committed end-of-log offsets for every override consumer group named
# in KAFKA_CONSUMER_GROUP, against every topic precreate_topics just created
# (design §6.3, FR-4.9). KAFKA_CONSUMER_GROUP is newline-delimited — group
# names contain spaces and brackets (e.g. "Account Service [pr-123]"), so a
# space-delimited list would be ambiguous.
#
# WHOEVER WIRES THIS: KAFKA_CONSUMER_GROUP must contain the RESOLVED,
# post-substitution group name(s) actually joined at runtime — never a raw
# literal copied from deploy/k8s/overlays/pr/patches/consumer-group-env.yaml.
# libs/atlas-kafka/consumergroup/resolver.go documents templated callers
# (atlas-login, atlas-channel) whose KAFKA_CONSUMER_GROUP is a format string
# such as "Channel Service - %s [pr-123]", substituted per channel/login
# instance at runtime — Resolve(defaultName, args...) does the %s fill-in,
# this script never does. Copying that literal in unresolved would seed a
# group name containing a literal "%s" that no runtime consumer ever joins:
# the seeding pass would report success and do nothing. A templated service
# needs one resolved entry per channel/login instance it actually runs, not
# the template string itself. See docs/runbooks/sparse-environments.md.
#
# Seeds the FULL topic union (both $topics and $compact_topics), not each
# group's true subscribed-topic subset — that data does not exist anywhere
# today (no service-to-topic subscription map), and deriving it would be a
# task of its own. This is a deliberate scope decision (task-232 Task 45
# review round 1, finding 3), not an oversight: a group carrying a committed
# offset on a topic it never reads is inert, and is reclaimed with the group
# when the environment tears down. With seed_group collapsed to one call per
# group (above), the cost of the superset is one extra --topic flag per
# unused topic, not one extra JVM start.
#
# Skips entirely when KAFKA_CONSUMER_GROUP is unset: that is main, whose
# groups already exist and carry real committed offsets that must not be
# reset (NG6). This is the FIRST thing the pass does, ahead of any Kafka
# call, so main is provably never touched by --reset-offsets.
#
# A group that is ALREADY ACTIVE is already initialized — which is the end
# state this pass exists to reach — so it is skipped, not reset (FR-1.3).
# This is what makes the pass idempotent across Argo CD re-syncs: the Job
# carries Force=true,Replace=true and is recreated on every sync, while the
# environment's Deployments have been joined to those groups for hours.
# Skipping is also the safety property (FR-5.2): an environment that has been
# consuming for hours must never be silently fast-forwarded past unprocessed
# messages by a routine re-sync.
seed_override_offsets() {
    if [ -z "${KAFKA_CONSUMER_GROUP:-}" ]; then
        echo "[$(date -u +%FT%TZ)] KAFKA_CONSUMER_GROUP unset — skipping override offset seeding (main, NG6)"
        return 0
    fi

    echo "[$(date -u +%FT%TZ)] seeding end-of-log offsets for override consumer groups"
    groups_file="$(mktemp)"
    printf '%s\n' "$KAFKA_CONSUMER_GROUP" > "$groups_file"
    all_topics="$(mktemp)"
    cat "$topics" "$compact_topics" > "$all_topics"
    # Shared with verify_group_offsets by the same globals-by-convention this
    # pass already uses for $topics / $compact_topics (FR-3.2). One group name
    # per line; membership is tested with grep -Fxq -- so a name containing
    # spaces, brackets or a leading dash matches exactly.
    skipped_groups="$(mktemp)"
    seeded_count=0
    skipped_count=0

    while IFS= read -r group; do
        [ -n "$group" ] || continue
        group_current_state="$(group_state "$group")"
        if state_is_seedable "$group_current_state"; then
            seed_rc=0
            # shellcheck disable=SC2046
            seed_group "$group" $(cat "$all_topics") || seed_rc=$?
            if [ "$seed_rc" -eq 0 ]; then
                seeded_count=$(( seeded_count + 1 ))
            elif [ "$seed_rc" -eq 2 ]; then
                # The window between the probe above and the reset is real: a
                # self-heal or a rescheduled pod can join the group in between
                # (FR-2.1). Same outcome as an FR-1.3 skip.
                printf '%s\n' "$group" >> "$skipped_groups"
                skipped_count=$(( skipped_count + 1 ))
                echo "[$(date -u +%FT%TZ)] skipping group '$group': reset refused, group became active during seeding — offsets already initialized"
            else
                echo "FAIL: seeding group '$group' failed (exit $seed_rc)" >&2
                exit "$seed_rc"
            fi
        else
            printf '%s\n' "$group" >> "$skipped_groups"
            skipped_count=$(( skipped_count + 1 ))
            echo "[$(date -u +%FT%TZ)] skipping group '$group': already active ($group_current_state) — offsets already initialized"
        fi
    done < "$groups_file"

    if [ "$seeded_count" -eq 0 ] && [ "$skipped_count" -gt 0 ]; then
        echo "[$(date -u +%FT%TZ)] all $skipped_count override consumer groups were already active — nothing seeded this run (re-sync no-op)"
    fi
    echo "[$(date -u +%FT%TZ)] override consumer group offsets seeded ($seeded_count seeded, $skipped_count skipped)"
}

# The activation gate (FR-5.3) needs an OBSERVABLE signal that a group is
# initialized, not an inference: --describe must report a committed offset
# (anything but "-") on every topic seed_override_offsets just seeded.
# Fails the whole Job (exit 1) if any is missing, so Argo CD's health check
# on this Job carries the signal. Symmetric skip with
# seed_override_offsets: nothing to verify when KAFKA_CONSUMER_GROUP is
# unset. Every seeded topic is single-partition (precreate_topics always
# creates --partitions 1), so checking one row per (group, topic) suffices.
# One --describe call per group (not per (group, topic) pair) already
# reports every topic/partition that group has offsets on.
#
# A group in $skipped_groups was skipped by seed_override_offsets because it
# is already ACTIVE — which means a live consumer is joined to it, which is
# the very state this gate exists to establish (FR-3.1). Re-proving it against
# the full topic union would fail the Job the first time a topic is added to a
# live environment. So for a skipped group the gate degrades to a report:
# union topics with no committed offset are named in a WARN: line and the Job
# stays green (an unseeded topic falls back to the consumer's own
# auto.offset.reset). For every other group the gate is unchanged and still
# exits 1 (FR-3.3). $skipped_groups may be unset — a test can source this file
# and call verify_group_offsets without seed_override_offsets — in which case
# nothing was skipped and every group takes the hard-gate path.
verify_group_offsets() {
    if [ -z "${KAFKA_CONSUMER_GROUP:-}" ]; then
        return 0
    fi

    echo "[$(date -u +%FT%TZ)] verifying override consumer group offsets are committed"
    groups_file="$(mktemp)"
    printf '%s\n' "$KAFKA_CONSUMER_GROUP" > "$groups_file"
    all_topics="$(mktemp)"
    cat "$topics" "$compact_topics" > "$all_topics"

    while IFS= read -r group; do
        [ -n "$group" ] || continue
        group_skipped=0
        if [ -n "${skipped_groups:-}" ] && [ -f "${skipped_groups:-}" ] && grep -Fxq -- "$group" "$skipped_groups"; then
            group_skipped=1
        fi
        described="$("$KAFKA_BIN/kafka-consumer-groups.sh" --bootstrap-server "$BOOTSTRAP_SERVERS" --group "$group" --describe 2>/dev/null || true)"
        topic_total=0
        missing_total=0
        missing_names=""
        while IFS= read -r topic; do
            [ -n "$topic" ] || continue
            topic_total=$(( topic_total + 1 ))
            # GROUP is column 1 in --describe output, but real group names
            # contain spaces (e.g. "Account Service [pr-123]"), which shift
            # every fixed column number — confirmed empirically (task-232
            # Task 45 review round 1 follow-up: an awk match on $2/$4 silently
            # never matched against a real multi-word group name, and this
            # verification pass would have failed the Job on every seeded
            # group). TOPIC through CLIENT-ID are always exactly 8
            # single-token trailing columns (topic names are plain
            # identifiers, never containing spaces), so anchor from the END
            # of the line via NF instead of a fixed forward offset: TOPIC is
            # NF-7, CURRENT-OFFSET is NF-5, regardless of how many
            # whitespace-separated tokens GROUP itself contributes.
            off="$(printf '%s\n' "$described" | awk -v t="$topic" 'NF>=9 && $(NF-7)==t {print $(NF-5)}' | head -n1)"
            if [ -z "$off" ] || [ "$off" = "-" ]; then
                if [ "$group_skipped" -eq 0 ]; then
                    echo "FAIL: group '$group' has no committed offset on topic '$topic'" >&2
                    exit 1
                fi
                missing_total=$(( missing_total + 1 ))
                # Bounded at 10 names plus a count (OQ-1): the union is ~170
                # topics and deliberately a superset, so a genuine gap list is
                # either tiny or large enough that the count is the signal.
                if [ "$missing_total" -le 10 ]; then
                    if [ -z "$missing_names" ]; then
                        missing_names="$topic"
                    else
                        missing_names="$missing_names, $topic"
                    fi
                fi
            fi
        done < "$all_topics"
        if [ "$group_skipped" -eq 1 ]; then
            if [ "$missing_total" -eq 0 ]; then
                echo "[$(date -u +%FT%TZ)] skipped group '$group': committed offsets present on all $topic_total topics"
            elif [ "$missing_total" -gt 10 ]; then
                echo "WARN: skipped group '$group' has no committed offset on $missing_total of $topic_total topics: $missing_names (+$(( missing_total - 10 )) more)" >&2
            else
                echo "WARN: skipped group '$group' has no committed offset on $missing_total of $topic_total topics: $missing_names" >&2
            fi
        fi
    done < "$groups_file"
    echo "[$(date -u +%FT%TZ)] override consumer group offsets verified"
}

main() {
    precreate_topics
    seed_override_offsets
    verify_group_offsets
}

# Run main only when this file is executed directly, not when it is sourced
# (e.g. by atlas-kafka-precreate_test.sh, which needs the function
# definitions without a live Kafka). `return` outside a function fails when
# the shell is executing the script directly, which the subshell probe below
# turns into a reliable executed-vs-sourced check under bash — the only
# interpreter this file is ever executed (not merely sourced) with.
if ! (return 0 2>/dev/null); then
    main "$@"
fi
