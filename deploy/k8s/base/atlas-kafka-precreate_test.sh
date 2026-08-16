#!/usr/bin/env sh
# deploy/k8s/base/atlas-kafka-precreate_test.sh
#
# The NG6 skip-guard assertion below never touches Kafka, so it always runs.
# The seed_group assertion requires a reachable Kafka; it SKIPs otherwise.
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck source=./kafka-precreate.sh
. "$SCRIPT_DIR/kafka-precreate.sh"

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

[ -n "${BOOTSTRAP_SERVERS:-}" ] || { echo "SKIP: BOOTSTRAP_SERVERS unset"; exit 0; }

TOPIC="atlas-precreate-test-$$"
GROUP="atlas-precreate-test-group-$$"
kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --create --topic "$TOPIC" --partitions 2

# Produce three messages BEFORE seeding: a correctly seeded group must not
# replay them.
printf 'a\nb\nc\n' | kafka-console-producer.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --topic "$TOPIC"

seed_group "$GROUP" "$TOPIC"   # the function under test, sourced from the Job script

for p in 0 1; do
    off=$(kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" \
            --group "$GROUP" --describe 2>/dev/null \
          | awk -v p="$p" '$3==p {print $4}')
    [ -n "$off" ] && [ "$off" != "-" ] \
        || { echo "FAIL: partition $p has no committed offset"; exit 1; }
done
echo "PASS"
