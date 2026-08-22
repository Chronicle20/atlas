// Package topics is the part of the tool that talks to the cluster
// controller: it creates the topic union in a single request, applies the
// compacted cleanup policy, waits for the client's metadata cache to know
// about every topic it just created, and reads end-of-log offsets — the one
// primitive both group phases build their offset-seeding math on.
package topics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"atlas.com/kafka-precreate/internal/discover"
	"atlas.com/kafka-precreate/internal/kafkaops"
)

const compactCleanupPolicy = "compact"

// EnsureResult tallies what Ensure did: how many topics it created new, and
// how many were already there.
type EnsureResult struct {
	Created  int
	Existing int
}

// Ensure creates every topic in t.Union() with one CreateTopics request, then
// applies cleanup.policy=compact to every topic in t.Compact with one
// IncrementalAlterConfigs request.
//
// There is deliberately no list-then-diff here (FR-2.5): CreateTopics is
// idempotent per topic on its own — a topic that already exists comes back
// as kafka.TopicAlreadyExists on that one entry, and every other topic in
// the same request still succeeds — so a pre-flight "does this topic already
// exist" list buys nothing except a second round trip. Diffing a list
// response against wanted names is exactly where the shell version's locale
// collation bug lived: `sort` under a non-C locale reordered topic names in
// a way string comparison downstream didn't expect, silently dropping
// topics from the diff. Not building the diff removes the class of bug, not
// just this instance of it.
func Ensure(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, t discover.Topics) (EnsureResult, error) {
	cfgs := make([]kafka.TopicConfig, 0, len(t.Plain)+len(t.Compact))
	for _, name := range t.Plain {
		cfgs = append(cfgs, kafka.TopicConfig{
			Topic:             name,
			NumPartitions:     1,
			ReplicationFactor: 1,
		})
	}
	for _, name := range t.Compact {
		cfgs = append(cfgs, kafka.TopicConfig{
			Topic:             name,
			NumPartitions:     1,
			ReplicationFactor: 1,
			ConfigEntries: []kafka.ConfigEntry{
				{ConfigName: "cleanup.policy", ConfigValue: compactCleanupPolicy},
			},
		})
	}

	resp, err := c.CreateTopics(ctx, &kafka.CreateTopicsRequest{Addr: addr, Topics: cfgs})
	if err != nil {
		return EnsureResult{}, fmt.Errorf("creating topics: %w", err)
	}

	var result EnsureResult
	var fatal []error
	for name, topicErr := range resp.Errors {
		switch {
		case topicErr == nil:
			result.Created++
		case errors.Is(topicErr, kafka.TopicAlreadyExists):
			result.Existing++
		default:
			fatal = append(fatal, fmt.Errorf("creating topic %q: %w", name, topicErr))
		}
	}
	if len(fatal) > 0 {
		return EnsureResult{}, errors.Join(fatal...)
	}

	if len(t.Compact) == 0 {
		return result, nil
	}

	// IncrementalAlterConfigs, not the legacy AlterConfigs: AlterConfigs is
	// full-replace for topic-level configs, so applying it here would reset
	// every other topic-level override (retention, segment size, whatever an
	// operator set by hand) back to its broker default (design §4 OQ-1).
	// Incremental set-only touches cleanup.policy and leaves the rest alone.
	//
	// This runs unconditionally over the whole compacted set, not just the
	// topics Ensure just created: a topic created before cleanup.policy was
	// part of this tool's policy is exactly the case FR-2.6 covers, and
	// setting a config that is already set to the same value is a no-op on
	// the broker.
	resources := make([]kafka.IncrementalAlterConfigsRequestResource, len(t.Compact))
	for i, name := range t.Compact {
		resources[i] = kafka.IncrementalAlterConfigsRequestResource{
			ResourceType: kafka.ResourceTypeTopic,
			ResourceName: name,
			Configs: []kafka.IncrementalAlterConfigsRequestConfig{
				{
					Name:            "cleanup.policy",
					Value:           compactCleanupPolicy,
					ConfigOperation: kafka.ConfigOperationSet,
				},
			},
		}
	}

	alterResp, err := c.IncrementalAlterConfigs(ctx, &kafka.IncrementalAlterConfigsRequest{Addr: addr, Resources: resources})
	if err != nil {
		return EnsureResult{}, fmt.Errorf("setting cleanup.policy=compact: %w", err)
	}

	var alterFatal []error
	for _, res := range alterResp.Resources {
		if res.Error != nil {
			alterFatal = append(alterFatal, fmt.Errorf("setting cleanup.policy=compact on topic %q: %w", res.ResourceName, res.Error))
		}
	}
	if len(alterFatal) > 0 {
		return EnsureResult{}, errors.Join(alterFatal...)
	}

	return result, nil
}

// SettleConfig controls the poll cadence and ceiling Settle applies.
type SettleConfig struct {
	Poll    time.Duration       // default 250ms
	Ceiling time.Duration       // default 30s
	Sleep   func(time.Duration) // nil ⇒ time.Sleep
	Now     func() time.Time    // nil ⇒ time.Now
}

// Settle polls Metadata until every name in names is present with at least
// one partition, and returns each topic's sorted-ascending partition IDs.
//
// This exists because kafka-go's client transport answers Metadata from a
// cluster-state cache it refreshes on Transport.MetadataTTL, not from a
// fresh broker round trip every time (transport.go:352-362 in the vendored
// v0.4.51 source). ListOffsets is split per topic-partition and each piece
// is routed to the partition's leader looked up in that same cache, so a
// topic the cache doesn't know about yet resolves to Broker{ID: -1} and the
// request for it fails. CreateTopics returning success only means the
// controller accepted the topic — it says nothing about when this client's
// own cached view of the cluster catches up. Without this loop, a
// first-sync run could create 170 topics and then fail to route offset
// reads or writes for a chunk of them purely on cache timing, not on
// anything actually wrong with the cluster.
func Settle(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, names []string, cfg SettleConfig) (map[string][]int, error) {
	if len(names) == 0 {
		return map[string][]int{}, nil
	}

	sleep := cfg.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	poll := cfg.Poll
	if poll == 0 {
		poll = 250 * time.Millisecond
	}
	ceiling := cfg.Ceiling
	if ceiling == 0 {
		ceiling = 30 * time.Second
	}

	start := now()
	var lastErr error

	for {
		resp, err := c.Metadata(ctx, &kafka.MetadataRequest{Addr: addr, Topics: names})
		if err != nil {
			lastErr = fmt.Errorf("fetching metadata: %w", err)
		} else {
			result := make(map[string][]int, len(names))
			for _, topic := range resp.Topics {
				if topic.Error != nil || len(topic.Partitions) == 0 {
					continue
				}
				ids := make([]int, len(topic.Partitions))
				for i, p := range topic.Partitions {
					ids[i] = p.ID
				}
				sort.Ints(ids)
				result[topic.Name] = ids
			}

			if len(result) == len(names) {
				return result, nil
			}
			lastErr = missingTopicsError(names, result)
		}

		if now().Sub(start)+poll > ceiling {
			return nil, lastErr
		}
		sleep(poll)
	}
}

func missingTopicsError(names []string, present map[string][]int) error {
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := present[name]; !ok {
			missing = append(missing, name)
		}
	}
	return fmt.Errorf("topics not yet visible in cluster metadata: %v", missing)
}

// EndOffsets fetches the end-of-log (last) offset for every (topic,
// partition) pair in partitions with a single ListOffsets request, and
// returns them keyed by topic then partition.
//
// A partition can appear in Metadata (which Settle already waited for)
// before it has an elected leader: partition visibility and leader election
// are different broker events. ListOffsets against such a partition returns
// kafka.NotLeaderForPartition or kafka.LeaderNotAvailable as a per-partition
// error inside an otherwise-successful response, so the whole request is
// re-issued (idempotent) under a bounded retry (retry) until either a
// leader is elected or the budget is exhausted. Every other per-partition
// error stays fatal on the first response that carries it.
func EndOffsets(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, partitions map[string][]int, retry kafkaops.RetryConfig) (map[string]map[int]int64, error) {
	if len(partitions) == 0 {
		return map[string]map[int]int64{}, nil
	}

	req := &kafka.ListOffsetsRequest{Addr: addr, Topics: make(map[string][]kafka.OffsetRequest, len(partitions))}
	for topic, ids := range partitions {
		reqs := make([]kafka.OffsetRequest, len(ids))
		for i, id := range ids {
			reqs[i] = kafka.LastOffsetOf(id)
		}
		req.Topics[topic] = reqs
	}

	var resp *kafka.ListOffsetsResponse
	err := kafkaops.WithLeaderRetry(ctx, retry, func() error {
		var innerErr error
		resp, innerErr = c.ListOffsets(ctx, req)
		if innerErr != nil {
			return innerErr
		}
		return firstLeaderError(partitions, resp)
	})
	if err != nil {
		return nil, fmt.Errorf("listing end offsets: %w", err)
	}

	result := make(map[string]map[int]int64, len(partitions))
	var fatal []error
	for topic, ids := range partitions {
		byPartition := make(map[int]kafka.PartitionOffsets, len(resp.Topics[topic]))
		for _, po := range resp.Topics[topic] {
			byPartition[po.Partition] = po
		}

		offsets := make(map[int]int64, len(ids))
		for _, id := range ids {
			po, ok := byPartition[id]
			if !ok {
				fatal = append(fatal, fmt.Errorf("end offset for topic %q partition %d missing from response", topic, id))
				continue
			}
			if po.Error != nil {
				fatal = append(fatal, fmt.Errorf("end offset for topic %q partition %d: %w", topic, id, po.Error))
				continue
			}
			offsets[id] = po.LastOffset
		}
		result[topic] = offsets
	}
	if len(fatal) > 0 {
		return nil, errors.Join(fatal...)
	}

	return result, nil
}

// firstLeaderError returns the first NotLeaderForPartition or
// LeaderNotAvailable per-partition error found in resp for the requested
// partitions, or nil if none is present. It exists so the fn passed to
// WithLeaderRetry can surface an element-level retriable error and drive
// the retry loop; every other per-partition error is left for the caller's
// own fatal handling once the loop returns.
func firstLeaderError(partitions map[string][]int, resp *kafka.ListOffsetsResponse) error {
	for topic, ids := range partitions {
		byPartition := make(map[int]kafka.PartitionOffsets, len(resp.Topics[topic]))
		for _, po := range resp.Topics[topic] {
			byPartition[po.Partition] = po
		}
		for _, id := range ids {
			po, ok := byPartition[id]
			if !ok || po.Error == nil {
				continue
			}
			if errors.Is(po.Error, kafka.NotLeaderForPartition) || errors.Is(po.Error, kafka.LeaderNotAvailable) {
				return fmt.Errorf("end offset for topic %q partition %d: %w", topic, id, po.Error)
			}
		}
	}
	return nil
}
