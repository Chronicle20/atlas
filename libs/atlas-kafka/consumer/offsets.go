package consumer

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// ReadEndOffsets returns the current high-water-mark offset for each
// partition of topic, keyed by partition id.
//
// CAUTION: do not gate a replay-from-first consumer's "caught up" on these
// raw offsets — a retention-purged partition has a high-water mark no such
// consumer can ever observe. Use ReadReplayableEndOffsets for that.
//
// Returns an empty map and no error for a topic that exists but has no
// partitions yet. Returns kafka.UnknownTopicOrPartition (as an error) when
// the topic is missing entirely.
func ReadEndOffsets(ctx context.Context, brokers []string, topic string) (map[int]int64, error) {
	return readOffsets(ctx, brokers, topic, func(first, last int64) int64 { return last })
}

// ReadReplayableEndOffsets returns, per partition, the end offset a
// FirstOffset consumer must reach to have replayed every retained record:
// the high-water mark while records are retained, or 0 once the log-start
// offset has caught up to it (retention purged everything, or the partition
// was never written). Projection caught-up gates treat 0 as trivially
// caught up, so a purged partition cannot wedge them waiting on offsets
// that no longer exist.
//
// Same topic-missing/empty semantics as ReadEndOffsets.
func ReadReplayableEndOffsets(ctx context.Context, brokers []string, topic string) (map[int]int64, error) {
	return readOffsets(ctx, brokers, topic, ReplayableEnd)
}

// ReplayableEnd collapses a partition's (log-start, high-water-mark) offset
// pair to the end offset a replay-from-first consumer can actually reach:
// 0 when nothing is retained (first >= last), the high-water mark otherwise.
func ReplayableEnd(first, last int64) int64 {
	if first >= last {
		return 0
	}
	return last
}

func readOffsets(ctx context.Context, brokers []string, topic string, collapse func(first, last int64) int64) (map[int]int64, error) {
	if len(brokers) == 0 {
		return nil, kafka.UnknownTopicOrPartition
	}
	d := &kafka.Dialer{Timeout: 10 * time.Second, DualStack: true}
	conn, err := d.DialContext(ctx, "tcp", brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	parts, err := conn.ReadPartitions(topic)
	if err != nil {
		return nil, err
	}

	out := make(map[int]int64, len(parts))
	for _, p := range parts {
		leader := net.JoinHostPort(p.Leader.Host, strconv.Itoa(p.Leader.Port))
		pc, err := d.DialLeader(ctx, "tcp", leader, topic, p.ID)
		if err != nil {
			return nil, err
		}
		first, last, err := pc.ReadOffsets()
		_ = pc.Close()
		if err != nil {
			return nil, err
		}
		out[p.ID] = collapse(first, last)
	}
	return out, nil
}
