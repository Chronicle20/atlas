// Package topics is the part of the tool that talks to the cluster
// controller: it creates the topic union in a single request, applies the
// compacted topic configuration, waits for the client's metadata cache to
// know about every topic it just created, and reads end-of-log offsets — the
// one primitive both group phases build their offset-seeding math on.
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

const (
	compactCleanupPolicy = "compact"

	// compactMaxCompactionLagMs bounds how long a record may sit
	// uncompacted. This is the knob that makes cleanup.policy=compact mean
	// anything: the cleaner never touches a partition's ACTIVE segment, so
	// a topic whose segment never rolls is never cleaned no matter what its
	// cleanup policy says. For a compacted topic the broker's effective
	// roll deadline is min(segment.ms, max.compaction.lag.ms), so setting
	// this to 10 minutes makes the segment roll ten minutes after its first
	// record — and the roll is what hands the cleaner something to work on.
	// Verified against apache/kafka:4.1.1 (design §2.2): a topic carrying
	// only this config rolled and compacted 200 records over 3 keys down to
	// 3, with no segment.ms set at all.
	compactMaxCompactionLagMs = "600000"

	// compactSegmentMs is the same 10-minute deadline expressed on the
	// policy-independent knob. It is deliberately equal to
	// compactMaxCompactionLagMs and is therefore redundant while the topic
	// is compacted (the broker takes the min of the two). It is set anyway
	// so the roll bound survives someone changing cleanup.policy, and so
	// the roll cadence is legible from a --describe without knowing the
	// min() rule. It is NOT the knob doing the work — deleting
	// max.compaction.lag.ms and keeping this one would not be equivalent
	// for a compacted topic.
	compactSegmentMs = "600000"

	// compactMinCleanableDirtyRatio lets the cleaner select a segment whose
	// dirty fraction is small, rather than the 0.5 default. This one was
	// not isolated by the live experiments (both ran at a dirty ratio near
	// 1.0) and the forced-cleaning path max.compaction.lag.ms drives
	// bypasses the ratio check anyway; it is cheap steady-state insurance,
	// inert-but-harmless rather than load-bearing.
	compactMinCleanableDirtyRatio = "0.01"
)

// compactTopicConfig is one (name, value) pair in the configuration every
// compacted topic must carry. It exists so the CreateTopics and
// IncrementalAlterConfigs request bodies are two projections of one
// declaration: a config applied at creation but not at alter (or the
// reverse) is precisely the defect this set was added to fix.
//
// It is a neutral local pair type rather than []kafka.ConfigEntry because
// the alter direction needs a third field (ConfigOperation) that
// kafka.ConfigEntry has no counterpart for — one of the two directions is a
// projection either way, so the canonical declaration keeps kafka types out
// of it and reads as policy.
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

// compactCreateEntries projects the declaration onto a CreateTopics
// per-topic config body. It builds a fresh slice per call:
// kafka.TopicConfig.ConfigEntries is a per-topic field kafka-go reads but
// does not document as immutable, and sharing one backing array across N
// resources would be a silent aliasing hazard for no measurable gain at
// these sizes.
func compactCreateEntries() []kafka.ConfigEntry {
	entries := make([]kafka.ConfigEntry, len(compactTopicConfigs))
	for i, cfg := range compactTopicConfigs {
		entries[i] = kafka.ConfigEntry{ConfigName: cfg.name, ConfigValue: cfg.value}
	}
	return entries
}

// compactAlterConfigs projects the same declaration onto an
// IncrementalAlterConfigs per-resource config body, set-only. Fresh slice
// per call, for the reason on compactCreateEntries.
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

// CompactConfigNames returns the names of the configs applied to every
// compacted topic, in declaration order. It exists so the alter-phase log
// line cannot drift from what was actually sent. Names only: the values are
// four short numbers recoverable from a --describe.
func CompactConfigNames() []string {
	names := make([]string, len(compactTopicConfigs))
	for i, cfg := range compactTopicConfigs {
		names[i] = cfg.name
	}
	return names
}

// EnsureResult tallies what Ensure did: how many topics it created new, and
// how many were already there.
type EnsureResult struct {
	Created  int
	Existing int
}

// Ensure creates every topic in t.Union() with one CreateTopics request, then
// applies the compacted topic configuration (cleanup.policy,
// max.compaction.lag.ms, segment.ms, min.cleanable.dirty.ratio) to every
// topic in t.Compact with one IncrementalAlterConfigs request.
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
			ConfigEntries:     compactCreateEntries(),
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
	//
	// The set is now four configs, not one. cleanup.policy=compact on its
	// own is inert: the cleaner never touches a partition's active
	// segment, so a topic whose segment never rolls is never cleaned. The
	// unconditional sweep is therefore also the repair path for topics
	// created by an earlier version of this tool — applying the new roll
	// bound makes their oversized active segment roll on the next append,
	// and the cleaner collapses it on its next pass (design §2.3).
	resources := make([]kafka.IncrementalAlterConfigsRequestResource, len(t.Compact))
	for i, name := range t.Compact {
		resources[i] = kafka.IncrementalAlterConfigsRequestResource{
			ResourceType: kafka.ResourceTypeTopic,
			ResourceName: name,
			Configs:      compactAlterConfigs(),
		}
	}

	alterResp, err := c.IncrementalAlterConfigs(ctx, &kafka.IncrementalAlterConfigsRequest{Addr: addr, Resources: resources})
	if err != nil {
		return EnsureResult{}, fmt.Errorf("applying compacted topic config: %w", err)
	}

	var alterFatal []error
	for _, res := range alterResp.Resources {
		if res.Error != nil {
			alterFatal = append(alterFatal, fmt.Errorf("applying compacted topic config on topic %q: %w", res.ResourceName, res.Error))
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
