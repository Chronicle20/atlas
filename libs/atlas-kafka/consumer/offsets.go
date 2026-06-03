package consumer

import (
	"context"
	"net"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// ReadEndOffsets returns the current high-water-mark offset for each
// partition of topic, keyed by partition id. It is used by projection
// caught-up gates: a subscriber snapshots these offsets at boot and is
// considered "ready" once it has consumed past every offset in the snapshot.
//
// Returns an empty map and no error for a topic that exists but has no
// partitions yet. Returns kafka.UnknownTopicOrPartition (as an error) when
// the topic is missing entirely.
func ReadEndOffsets(ctx context.Context, brokers []string, topic string) (map[int]int64, error) {
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
		_, last, err := pc.ReadOffsets()
		_ = pc.Close()
		if err != nil {
			return nil, err
		}
		out[p.ID] = last
	}
	return out, nil
}
