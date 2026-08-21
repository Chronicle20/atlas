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
CONSUMER_CMD="${KAFKA_BIN:+$KAFKA_BIN/}kafka-console-consumer.sh"

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

# FR-5.2 — the safety property. A sparse environment that has been consuming
# for hours must not be silently fast-forwarded past unprocessed messages by
# a routine Argo CD re-sync. Kafka refuses --reset-offsets on an active group,
# but MEASURED (design §2.3) it refuses with EXIT 0 and an Error: line on
# stdout, so before this change the refusal was a silent no-op reporting
# success. seed_group must now report that as return code 2, without aborting
# the caller under set -eu, and without moving the committed offset.
#
# The group is held active by a consumer on a DIFFERENT topic ($TOPIC_D) than
# the one under test ($TOPIC_C): kafka-console-consumer auto-commits, so a
# consumer on $TOPIC_C would move the very offset this test asserts is
# unchanged. Committed offsets persist independently of subscription, so the
# seeded offset on $TOPIC_C survives untouched.
TOPIC_C="atlas-precreate-test-c-$$"
TOPIC_D="atlas-precreate-test-d-$$"
TOPIC_E="atlas-precreate-test-e-$$"
GROUP3="atlas-precreate-test-group3-$$"
for t in "$TOPIC_C" "$TOPIC_D" "$TOPIC_E"; do
    "$TOPICS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --create --topic "$t" --partitions 1
done
printf 'a\nb\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_C"
printf 'x\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_D"

seed_group "$GROUP3" "$TOPIC_C"

committed_offset() {
    "$CONSUMER_GROUPS_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --group "$1" --describe 2>/dev/null \
        | awk -v t="$2" 'NF>=9 && $(NF-7)==t {print $(NF-5)}' | head -n1
}

OFF_BEFORE="$(committed_offset "$GROUP3" "$TOPIC_C")"
if [ -z "$OFF_BEFORE" ] || [ "$OFF_BEFORE" = "-" ]; then
    echo "FAIL: setup — seed_group left '$TOPIC_C' without a committed offset"
    exit 1
fi

"$CONSUMER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" \
    --group "$GROUP3" --topic "$TOPIC_D" --from-beginning >/dev/null 2>&1 &
CONSUMER_PID=$!
# Never leave a consumer attached to the shared broker on a mid-test failure.
trap 'kill "$CONSUMER_PID" 2>/dev/null || true' EXIT INT TERM

# Bounded poll — a hard attempt cap, never an unbounded while.
STATE_NOW=""
attempt=0
while [ "$attempt" -lt 30 ]; do
    STATE_NOW="$(group_state "$GROUP3")"
    [ "$STATE_NOW" = "Stable" ] && break
    attempt=$(( attempt + 1 ))
    sleep 2
done
if [ "$STATE_NOW" != "Stable" ]; then
    echo "FAIL: group '$GROUP3' did not reach Stable within 30 attempts (last state: '$STATE_NOW')"
    exit 1
fi
echo "PASS: group_state reports 'Stable' for a group with a live member"

# A successful --to-latest reset would now move $TOPIC_C's offset from 2 to 5.
printf 'd\ne\nf\n' | "$PRODUCER_CMD" --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC_C"

if state_is_seedable "$STATE_NOW"; then
    echo "FAIL: state_is_seedable called '$STATE_NOW' seedable"
    exit 1
fi

seed_rc3=0
seed_group "$GROUP3" "$TOPIC_C" || seed_rc3=$?
if [ "$seed_rc3" -ne 2 ]; then
    echo "FAIL: seed_group against an active group returned $seed_rc3, expected 2"
    exit 1
fi
echo "PASS: seed_group returns 2 for an active group without aborting under set -eu"

OFF_AFTER="$(committed_offset "$GROUP3" "$TOPIC_C")"
if [ "$OFF_AFTER" != "$OFF_BEFORE" ]; then
    echo "FAIL: active group's committed offset moved $OFF_BEFORE -> $OFF_AFTER (FR-5.2)"
    exit 1
fi
echo "PASS: seed_group did not move an active group's committed offset (FR-5.2)"

# FR-5.1 — running the pass twice against the same live environment exits 0
# both times, and (FR-5.2) changes no committed offset. Also the FR-3.1
# regression test: $TOPIC_E is in the union and $GROUP3 has no committed
# offset on it, which before this change made verify_group_offsets exit 1 and
# fail the whole Job.
#
# main is not driven directly because precreate_topics uses compgen (bash) and
# this harness is #!/usr/bin/env sh; $topics / $compact_topics are the globals
# precreate_topics would otherwise set, so setting them here exercises the
# same seam.
topics="$(mktemp)"
compact_topics="$(mktemp)"
printf '%s\n%s\n' "$TOPIC_C" "$TOPIC_E" > "$topics"
: > "$compact_topics"
KAFKA_CONSUMER_GROUP="$GROUP3"
export KAFKA_CONSUMER_GROUP

# Command substitution is a subshell, so verify_group_offsets' exit 1 fails
# the substitution instead of killing this script — which is what lets the
# assertion report rather than merely die.
if ! pass1="$( { seed_override_offsets; verify_group_offsets; } 2>&1 )"; then
    echo "FAIL: first seed+verify pass against an active group did not exit 0"
    printf '%s\n' "$pass1"
    exit 1
fi
case "$pass1" in
    *"already active (Stable)"*) ;;
    *) echo "FAIL: first pass did not log the active-group skip"; printf '%s\n' "$pass1"; exit 1 ;;
esac
case "$pass1" in
    *"nothing seeded this run (re-sync no-op)"*) ;;
    *) echo "FAIL: first pass did not log the all-skipped re-sync no-op line"; printf '%s\n' "$pass1"; exit 1 ;;
esac
case "$pass1" in
    *"(0 seeded, 1 skipped)"*) ;;
    *) echo "FAIL: first pass did not report the seeded/skipped counts"; printf '%s\n' "$pass1"; exit 1 ;;
esac
case "$pass1" in
    *"WARN: skipped group '$GROUP3' has no committed offset on 1 of 2 topics: $TOPIC_E"*) ;;
    *) echo "FAIL: first pass did not WARN about the unseeded union topic"; printf '%s\n' "$pass1"; exit 1 ;;
esac
echo "PASS: seed+verify skips an active group, warns on its unseeded topic, and exits 0 (FR-3.1)"

if ! pass2="$( { seed_override_offsets; verify_group_offsets; } 2>&1 )"; then
    echo "FAIL: second seed+verify pass did not exit 0 (FR-5.1)"
    printf '%s\n' "$pass2"
    exit 1
fi
echo "PASS: a second full pass against a live environment exits 0 (FR-5.1)"

OFF_FINAL="$(committed_offset "$GROUP3" "$TOPIC_C")"
if [ "$OFF_FINAL" != "$OFF_BEFORE" ]; then
    echo "FAIL: committed offset moved $OFF_BEFORE -> $OFF_FINAL across two passes (FR-5.2)"
    exit 1
fi
echo "PASS: two full passes changed no committed offset (FR-5.2)"
