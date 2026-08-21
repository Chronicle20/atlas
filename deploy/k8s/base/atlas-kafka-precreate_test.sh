#!/usr/bin/env sh
# deploy/k8s/base/atlas-kafka-precreate_test.sh
#
# The NG6 skip-guard assertion below never touches Kafka, so it always runs.
# The seed_group assertion requires a reachable Kafka; it SKIPs otherwise.
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck source=./kafka-precreate.sh
. "$SCRIPT_DIR/kafka-precreate.sh"

# Same PATH problem kafka-precreate.sh's KAFKA_BIN solves (task-232 Task 45
# review round 1, Critical 1): apache/kafka:3.7.2's PATH does not include
# the Kafka CLI tools, only /opt/kafka/bin has them. Default to that
# directory so this test runs against the real image with no PATH
# injection; fall back to bare names (resolved via the caller's own PATH)
# when /opt/kafka/bin doesn't exist, e.g. a dev machine with the CLI tools
# installed elsewhere. Still overridable — set KAFKA_BIN before invoking to
# point at a different install.
KAFKA_BIN="${KAFKA_BIN:-/opt/kafka/bin}"
[ -d "$KAFKA_BIN" ] || KAFKA_BIN=""
TOPICS_CMD="${KAFKA_BIN:+$KAFKA_BIN/}kafka-topics.sh"
PRODUCER_CMD="${KAFKA_BIN:+$KAFKA_BIN/}kafka-console-producer.sh"
CONSUMER_GROUPS_CMD="${KAFKA_BIN:+$KAFKA_BIN/}kafka-consumer-groups.sh"

# NG6: main's groups already exist and carry real committed offsets, so
# --reset-offsets --to-latest --execute against them would skip unconsumed
# backlog in production. seed_override_offsets must be a no-op whenever
# KAFKA_CONSUMER_GROUP is unset (i.e. always, on main) — asserted without
# touching Kafka, so this check runs even when no broker is reachable.
unset KAFKA_CONSUMER_GROUP 2>/dev/null || true
skip_out="$(seed_override_offsets)"
case "$skip_out" in
    *"KAFKA_CONSUMER_GROUP unset"*) ;;
    *)
        echo "FAIL: seed_override_offsets did not skip when KAFKA_CONSUMER_GROUP is unset"
        exit 1
        ;;
esac
echo "PASS: seed_override_offsets skips when KAFKA_CONSUMER_GROUP is unset (NG6)"

# state_is_seedable is an ALLOWLIST, not a denylist (design §3.1): only
# Empty, Dead and the empty string (absent group / unparseable output /
# failed probe, FR-1.4) are seedable. Anything else — including a state
# token Kafka adds in a future version — is treated as active and skipped,
# because skipping never mutates a committed offset (FR-5.2) whereas
# resetting an unrecognised live state does. Asserted without a broker, so
# this contract is enforced on every run of this script.
for state in Empty Dead ""; do
    if ! state_is_seedable "$state"; then
        echo "FAIL: state_is_seedable rejected seedable state '$state'"
        exit 1
    fi
done
for state in Stable PreparingRebalance CompletingRebalance SomeNewState STATE; do
    if state_is_seedable "$state"; then
        echo "FAIL: state_is_seedable accepted active state '$state'"
        exit 1
    fi
done
echo "PASS: state_is_seedable allowlists Empty/Dead/unknown and rejects every active state"

[ -n "${BOOTSTRAP_SERVERS:-}" ] || { echo "SKIP: BOOTSTRAP_SERVERS unset"; exit 0; }

TOPIC="atlas-precreate-test-$$"
GROUP="atlas-precreate-test-group-$$"
"$TOPICS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --create --topic "$TOPIC" --partitions 2

# Produce three messages BEFORE seeding: a correctly seeded group must not
# replay them.
printf 'a\nb\nc\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC"

seed_group "$GROUP" "$TOPIC"   # the function under test, sourced from the Job script

for p in 0 1; do
    off=$("$CONSUMER_GROUPS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --group "$GROUP" --describe 2>/dev/null \
          | awk -v p="$p" '$3==p {print $4}')
    [ -n "$off" ] && [ "$off" != "-" ] \
        || { echo "FAIL: partition $p has no committed offset"; exit 1; }
done
echo "PASS"

# seed_group now takes a group plus one-or-more topics and collapses them
# into a SINGLE kafka-consumer-groups.sh --reset-offsets --execute call with
# repeated --topic flags (task-232 Task 45 review round 1, finding 2) — the
# fix for the O(groups x topics) JVM-cold-start cost of one invocation per
# (group, topic) pair. Assert that a single multi-topic call actually seeds
# every named topic, not just the last one.
TOPIC_A="atlas-precreate-test-a-$$"
TOPIC_B="atlas-precreate-test-b-$$"
GROUP2="atlas-precreate-test-group2-$$"
"$TOPICS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --create --topic "$TOPIC_A" --partitions 1
"$TOPICS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --create --topic "$TOPIC_B" --partitions 1
printf 'a\nb\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_A"
printf 'a\nb\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_B"

seed_group "$GROUP2" "$TOPIC_A" "$TOPIC_B"

described="$("$CONSUMER_GROUPS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --group "$GROUP2" --describe 2>/dev/null)"
for t in "$TOPIC_A" "$TOPIC_B"; do
    off=$(printf '%s\n' "$described" | awk -v t="$t" '$2==t {print $4}')
    if [ -z "$off" ] || [ "$off" = "-" ]; then
        echo "FAIL: multi-topic seed_group left '$t' without a committed offset"
        exit 1
    fi
done
echo "PASS: seed_group seeds every topic in a single multi-topic call"
