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

    # Each kafka-topics.sh --create is a full JVM cold start (~1-1.5s) and only
    # creates one topic per call, so a serial loop over ~140 topics takes minutes.
    # Fan out with xargs -P so the JVM starts overlap; --if-not-exists keeps it
    # idempotent and concurrent creates of distinct topics don't conflict. xargs
    # exits non-zero if any child fails, which set -e turns into a Job retry.
    echo "[$(date -u +%FT%TZ)] creating $count topics (concurrency 16)"
    xargs -P 16 -I {} \
        /opt/kafka/bin/kafka-topics.sh \
            --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --create --if-not-exists \
            --topic {} \
            --partitions 1 \
            --replication-factor 1 \
        < "$topics" >/dev/null

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
        # --create --if-not-exists skips existing topics, so also
        # alter (idempotent) to converge topics created before this
        # policy existed.
        /opt/kafka/bin/kafka-configs.sh \
            --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --alter --entity-type topics --entity-name "$topic" \
            --add-config cleanup.policy=compact \
            >/dev/null
    done < "$compact_topics"

    echo "[$(date -u +%FT%TZ)] reconciled $count topics"
}

# Commit end-of-log offsets for an override consumer group on one topic,
# while the group is empty and therefore resettable. Runs at sync-wave 0,
# before any Deployment starts (design §6.3).
seed_group() {
    group="$1"; topic="$2"
    kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" \
        --group "$group" --topic "$topic" --reset-offsets --to-latest --execute >/dev/null
}

# Seeds committed end-of-log offsets for every override consumer group named
# in KAFKA_CONSUMER_GROUP, against every topic precreate_topics just created
# (design §6.3, FR-4.9). KAFKA_CONSUMER_GROUP is newline-delimited — group
# names contain spaces and brackets (e.g. "Account Service [pr-123]"), so a
# space-delimited list would be ambiguous.
#
# Skips entirely when KAFKA_CONSUMER_GROUP is unset: that is main, whose
# groups already exist and carry real committed offsets that must not be
# reset (NG6). This is the FIRST thing the pass does, ahead of any Kafka
# call, so main is provably never touched by --reset-offsets.
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

    while IFS= read -r group; do
        [ -n "$group" ] || continue
        while IFS= read -r topic; do
            [ -n "$topic" ] || continue
            seed_group "$group" "$topic"
        done < "$all_topics"
    done < "$groups_file"
    echo "[$(date -u +%FT%TZ)] override consumer group offsets seeded"
}

# The activation gate (FR-5.3) needs an OBSERVABLE signal that a group is
# initialized, not an inference: --describe must report a committed offset
# (anything but "-") on every topic seed_override_offsets just seeded.
# Fails the whole Job (exit 1) if any is missing, so Argo CD's health check
# on this Job carries the signal. Symmetric skip with
# seed_override_offsets: nothing to verify when KAFKA_CONSUMER_GROUP is
# unset. Every seeded topic is single-partition (precreate_topics always
# creates --partitions 1), so checking one row per (group, topic) suffices.
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
        described="$(kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --group "$group" --describe 2>/dev/null || true)"
        while IFS= read -r topic; do
            [ -n "$topic" ] || continue
            off="$(printf '%s\n' "$described" | awk -v t="$topic" '$2==t {print $4}' | head -n1)"
            if [ -z "$off" ] || [ "$off" = "-" ]; then
                echo "FAIL: group '$group' has no committed offset on topic '$topic'" >&2
                exit 1
            fi
        done < "$all_topics"
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
