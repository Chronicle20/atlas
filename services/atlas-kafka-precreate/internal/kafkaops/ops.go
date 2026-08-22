// Package kafkaops defines the narrow Kafka admin interface this tool is
// written against, and the bounded retry it applies to group-coordinator
// requests.
package kafkaops

import (
	"context"
	"errors"
	"fmt"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

// AdminClient is the set of kafka-go Client operations this tool needs.
// *kafka.Client satisfies it with no adapter.
type AdminClient interface {
	CreateTopics(context.Context, *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error)
	IncrementalAlterConfigs(context.Context, *kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error)
	Metadata(context.Context, *kafka.MetadataRequest) (*kafka.MetadataResponse, error)
	ListOffsets(context.Context, *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error)
	DescribeGroups(context.Context, *kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error)
	OffsetCommit(context.Context, *kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error)
	OffsetFetch(context.Context, *kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error)
}

var _ AdminClient = (*kafka.Client)(nil)

// RetryConfig controls the bounded backoff WithCoordinatorRetry applies.
type RetryConfig struct {
	Base   time.Duration       // first backoff; default 250ms
	Max    time.Duration       // per-attempt cap; default 2s
	Budget time.Duration       // total wall-clock budget; default 60s
	Sleep  func(time.Duration) // nil ⇒ time.Sleep
	Now    func() time.Time    // nil ⇒ time.Now
}

// DefaultRetryConfig returns the production defaults: 250ms base backoff,
// doubling to a 2s per-attempt cap, within a 60s total budget.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Base:   250 * time.Millisecond,
		Max:    2 * time.Second,
		Budget: 60 * time.Second,
	}
}

// isCoordinatorError reports whether err is one of the two transient
// group-coordinator errors that WithCoordinatorRetry retries.
func isCoordinatorError(err error) bool {
	return errors.Is(err, kafka.NotCoordinatorForGroup) || errors.Is(err, kafka.GroupCoordinatorNotAvailable)
}

// isLeaderError reports whether err is one of the two transient
// partition-leader errors that WithLeaderRetry retries.
func isLeaderError(err error) bool {
	return errors.Is(err, kafka.NotLeaderForPartition) || errors.Is(err, kafka.LeaderNotAvailable)
}

// WithCoordinatorRetry calls fn, retrying with exponential backoff (capped
// at cfg.Max, bounded by cfg.Budget) only when fn returns
// kafka.NotCoordinatorForGroup or kafka.GroupCoordinatorNotAvailable — the
// two codes a client can see while the group coordinator is being
// (re)elected.
//
// This wraps the three group-coordinator calls: DescribeGroups,
// OffsetCommit, and OffsetFetch. CreateTopics, Metadata, and
// IncrementalAlterConfigs route to the controller and cannot produce these
// two codes; retrying anything else would turn a diagnosable failure into a
// timeout (design §4 OQ-3).
func WithCoordinatorRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	return withRetry(ctx, cfg, isCoordinatorError, fn)
}

// WithLeaderRetry calls fn, retrying with exponential backoff (capped at
// cfg.Max, bounded by cfg.Budget) only when fn returns
// kafka.NotLeaderForPartition or kafka.LeaderNotAvailable — the two codes a
// client can see for a partition immediately after CreateTopics, before
// leader election has completed. Partition visibility in Metadata is not
// leader election, so ListOffsets against a just-created partition can
// return these codes even though the topic itself was created successfully.
//
// This wraps ListOffsets. Retrying anything else would turn a diagnosable
// failure into a timeout (design §4 OQ-3).
func WithLeaderRetry(ctx context.Context, cfg RetryConfig, fn func() error) error {
	return withRetry(ctx, cfg, isLeaderError, fn)
}

// withRetry calls fn, retrying with exponential backoff (capped at cfg.Max,
// bounded by cfg.Budget) only when fn returns an error for which retriable
// reports true.
func withRetry(ctx context.Context, cfg RetryConfig, retriable func(error) bool, fn func() error) error {
	sleep := cfg.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	defaults := DefaultRetryConfig()
	base := cfg.Base
	if base == 0 {
		base = defaults.Base
	}
	max := cfg.Max
	if max == 0 {
		max = defaults.Max
	}
	budget := cfg.Budget
	if budget == 0 {
		budget = defaults.Budget
	}

	start := now()
	backoff := base

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := fn()
		if err == nil {
			return nil
		}
		if !retriable(err) {
			return err
		}

		if now().Sub(start)+backoff > budget {
			return fmt.Errorf("retry budget of %s exhausted: %w", budget, err)
		}

		sleep(backoff)
		backoff *= 2
		if backoff > max {
			backoff = max
		}
	}
}
