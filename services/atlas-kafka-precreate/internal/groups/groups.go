// Package groups is the group-coordinator half of the precreate tool: it
// probes each override consumer group's state, seeds end-of-log offsets for
// the groups that are safe to seed, and runs the asymmetric verification
// gate that turns "offsets were committed" into an observable signal.
package groups

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"

	kafka "github.com/segmentio/kafka-go"

	"atlas.com/kafka-precreate/internal/discover"
	"atlas.com/kafka-precreate/internal/kafkaops"
)

// SeedResult tallies what Seed did to each group in groupIDs.
type SeedResult struct {
	Seeded  []string          // groups seeded this run, input order
	Skipped []string          // groups skipped (active state, or the commit race)
	States  map[string]string // group → observed state, for logging
}

// WasSkipped reports whether group appears in r.Skipped.
func (r SeedResult) WasSkipped(group string) bool {
	for _, g := range r.Skipped {
		if g == group {
			return true
		}
	}
	return false
}

// Seed seeds end-of-log offsets, as a non-member (GenerationID -1, empty
// MemberID), for every group in groupIDs that is currently safe to seed.
//
// An empty groupIDs returns a zero SeedResult before any Kafka call. This
// is NG6: main's groupIDs (KAFKA_CONSUMER_GROUP unset) is the empty slice,
// and main's groups carry real committed offsets that must never be reset.
// Being the first statement makes "main is provably never touched" a
// property of this code, not of the caller that happens to pass it an
// empty slice today.
func Seed(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, groupIDs []string,
	partitions map[string][]int, offsets map[string]map[int]int64,
	retry kafkaops.RetryConfig,
) (SeedResult, error) {
	if len(groupIDs) == 0 {
		return SeedResult{}, nil
	}

	result := SeedResult{States: make(map[string]string, len(groupIDs))}

	for _, group := range groupIDs {
		state := probeState(ctx, c, addr, group, retry)
		result.States[group] = state

		if !discover.StateIsSeedable(state) {
			result.Skipped = append(result.Skipped, group)
			continue
		}

		topics := make(map[string][]kafka.OffsetCommit, len(partitions))
		for topic, ids := range partitions {
			commits := make([]kafka.OffsetCommit, 0, len(ids))
			for _, p := range ids {
				commits = append(commits, kafka.OffsetCommit{
					Partition: p,
					Offset:    offsets[topic][p],
				})
			}
			topics[topic] = commits
		}

		var resp *kafka.OffsetCommitResponse
		err := kafkaops.WithCoordinatorRetry(ctx, retry, func() error {
			var innerErr error
			resp, innerErr = c.OffsetCommit(ctx, &kafka.OffsetCommitRequest{
				Addr:         addr,
				GroupID:      group,
				GenerationID: -1,
				MemberID:     "",
				Topics:       topics,
			})
			return innerErr
		})
		if err != nil {
			return SeedResult{}, fmt.Errorf("committing seed offsets for group %q: %w", group, err)
		}

		// Code 25 (kafka.UnknownMemberId) is not literally "this group is
		// active" — it is the broker's generic "no member with this ID in
		// this generation" response. Because this tool always commits with
		// MemberID "" and GenerationID -1, though, the only way a commit
		// can come back UnknownMemberId is that a real member joined the
		// group between the DescribeGroups probe above and this commit,
		// pushing the group past generation -1. That race window is why
		// the probe stays a probe rather than being replaced by "just
		// commit and see": without it, every commit would need this same
		// error-triage logic with no state context to make it unambiguous.
		commitRace := false
		var fatal error
		for topic, parts := range resp.Topics {
			for _, part := range parts {
				if part.Error == nil {
					continue
				}
				if errors.Is(part.Error, kafka.UnknownMemberId) {
					commitRace = true
					break
				}
				fatal = fmt.Errorf("committing seed offset for group %q topic %q partition %d: %w",
					group, topic, part.Partition, part.Error)
				break
			}
			if commitRace || fatal != nil {
				break
			}
		}
		if fatal != nil {
			return SeedResult{}, fatal
		}
		if commitRace {
			result.Skipped = append(result.Skipped, group)
			continue
		}

		result.Seeded = append(result.Seeded, group)
	}

	return result, nil
}

// probeState returns the group's observed state, collapsing a transport
// error, a per-group Error, or an absent group to "" — all three read as
// "not active" to StateIsSeedable, and the probe must never itself fail the
// Job (it exists only to decide seedable vs. skip).
func probeState(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, group string, retry kafkaops.RetryConfig) string {
	var resp *kafka.DescribeGroupsResponse
	err := kafkaops.WithCoordinatorRetry(ctx, retry, func() error {
		var innerErr error
		resp, innerErr = c.DescribeGroups(ctx, &kafka.DescribeGroupsRequest{
			Addr:     addr,
			GroupIDs: []string{group},
		})
		return innerErr
	})
	if err != nil || resp == nil {
		return ""
	}
	for _, g := range resp.Groups {
		if g.GroupID != group {
			continue
		}
		if g.Error != nil {
			return ""
		}
		return g.GroupState
	}
	return ""
}

// VerifyReport is the outcome of checking one group's committed offsets
// against the requested (topic, partition) union.
type VerifyReport struct {
	Group   string
	Total   int      // union (topic, partition) pairs checked
	Missing []string // topic names with at least one uncommitted partition, sorted
}

// Verify checks, per group, that every (topic, partition) in partitions
// carries a committed offset.
//
// For a group in seeded.Skipped, a missing offset is reported (WARN
// material), not failed: a group skipped by Seed was already active, which
// means a live consumer is joined to it — the very end state this gate
// exists to establish. Re-proving it against the full topic union would
// fail the Job the first time a topic is added to a live environment, so
// for a skipped group the gate degrades to a report and the Job stays
// green (an unseeded topic falls back to the consumer's own
// auto.offset.reset). For every other (seeded) group the gate is
// unchanged: the first missing offset is a fatal error naming the group
// and topic.
//
// A top-level OffsetFetchResponse.Error is fatal regardless of seeded vs.
// skipped: an RPC-level failure (e.g. an authorization failure) is not the
// same thing as a missing offset, and warning on it would hide a real
// misconfiguration.
func Verify(ctx context.Context, c kafkaops.AdminClient, addr net.Addr, groupIDs []string,
	partitions map[string][]int, seeded SeedResult,
	retry kafkaops.RetryConfig,
) ([]VerifyReport, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}

	reports := make([]VerifyReport, 0, len(groupIDs))

	for _, group := range groupIDs {
		var resp *kafka.OffsetFetchResponse
		err := kafkaops.WithCoordinatorRetry(ctx, retry, func() error {
			var innerErr error
			resp, innerErr = c.OffsetFetch(ctx, &kafka.OffsetFetchRequest{
				Addr:    addr,
				GroupID: group,
				Topics:  partitions,
			})
			return innerErr
		})
		if err != nil {
			return nil, fmt.Errorf("fetching committed offsets for group %q: %w", group, err)
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("fetching committed offsets for group %q: %w", group, resp.Error)
		}

		skipped := seeded.WasSkipped(group)
		total := 0
		missing := make(map[string]struct{})

		for _, topic := range sortedKeys(partitions) {
			ids := partitions[topic]
			committed := make(map[int]kafka.OffsetFetchPartition, len(ids))
			for _, part := range resp.Topics[topic] {
				committed[part.Partition] = part
			}

			for _, p := range ids {
				total++
				part, ok := committed[p]
				if !ok {
					missing[topic] = struct{}{}
					continue
				}
				if part.Error != nil {
					return nil, fmt.Errorf("fetching committed offset for group %q topic %q partition %d: %w",
						group, topic, p, part.Error)
				}
				if part.CommittedOffset < 0 {
					missing[topic] = struct{}{}
				}
			}
		}

		if len(missing) > 0 && !skipped {
			for _, topic := range sortedKeys(partitions) {
				if _, ok := missing[topic]; ok {
					return nil, fmt.Errorf("group %q has no committed offset on topic %q", group, topic)
				}
			}
		}

		names := make([]string, 0, len(missing))
		for topic := range missing {
			names = append(names, topic)
		}
		sort.Strings(names)

		reports = append(reports, VerifyReport{Group: group, Total: total, Missing: names})
	}

	return reports, nil
}

func sortedKeys(m map[string][]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
