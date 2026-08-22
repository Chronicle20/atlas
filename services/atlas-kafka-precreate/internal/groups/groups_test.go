package groups

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"atlas.com/kafka-precreate/internal/kafkaops"
)

// stubClient implements kafkaops.AdminClient for tests in this package. The
// three group-coordinator methods record every call and delegate to their
// *Fn field, returning a zero-value response when unset. The other four
// methods panic: groups must never issue them, and the panic is the proof.
type stubClient struct {
	describeCalls []*kafka.DescribeGroupsRequest
	describeFn    func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error)

	commitCalls []*kafka.OffsetCommitRequest
	commitFn    func(*kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error)

	fetchCalls []*kafka.OffsetFetchRequest
	fetchFn    func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error)
}

func (s *stubClient) CreateTopics(context.Context, *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
	panic("unexpected call")
}

func (s *stubClient) IncrementalAlterConfigs(context.Context, *kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error) {
	panic("unexpected call")
}

func (s *stubClient) Metadata(context.Context, *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
	panic("unexpected call")
}

func (s *stubClient) ListOffsets(context.Context, *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
	panic("unexpected call")
}

func (s *stubClient) DescribeGroups(_ context.Context, req *kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
	s.describeCalls = append(s.describeCalls, req)
	if s.describeFn == nil {
		return &kafka.DescribeGroupsResponse{}, nil
	}
	return s.describeFn(req)
}

func (s *stubClient) OffsetCommit(_ context.Context, req *kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error) {
	s.commitCalls = append(s.commitCalls, req)
	if s.commitFn == nil {
		return &kafka.OffsetCommitResponse{}, nil
	}
	return s.commitFn(req)
}

func (s *stubClient) OffsetFetch(_ context.Context, req *kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
	s.fetchCalls = append(s.fetchCalls, req)
	if s.fetchFn == nil {
		return &kafka.OffsetFetchResponse{}, nil
	}
	return s.fetchFn(req)
}

var _ kafkaops.AdminClient = (*stubClient)(nil)

const chanGroup = "Channel Service - 7c2f8b1e-0d4a-4a1b-9f3e-2c1d5e6f7a8b [pr-1450]"

func noWaitRetry() kafkaops.RetryConfig {
	return kafkaops.RetryConfig{
		Base:   time.Millisecond,
		Max:    time.Millisecond,
		Budget: 10 * time.Millisecond,
		Sleep:  func(time.Duration) {},
		Now:    time.Now,
	}
}

func sharedFixtures() (map[string][]int, map[string]map[int]int64) {
	partitions := map[string][]int{"a": {0}, "b": {0}}
	offsets := map[string]map[int]int64{"a": {0: 5}, "b": {0: 0}}
	return partitions, offsets
}

func mustContain(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}

func TestSeed(t *testing.T) {
	partitions, offsets := sharedFixtures()

	cases := []struct {
		name            string
		groups          []string
		describeFn      func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error)
		commitFn        func(*kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error)
		wantSeeded      []string
		wantSkipped     []string
		wantStates      map[string]string
		wantErr         string
		wantCommitCalls int
		checkCommit     func(t *testing.T, calls []*kafka.OffsetCommitRequest)
	}{
		{
			name:            "empty group list",
			groups:          []string{},
			wantSeeded:      nil,
			wantSkipped:     nil,
			wantCommitCalls: 0,
		},
		{
			name:   "seedable Empty group",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "Empty"},
				}}, nil
			},
			wantSeeded:      []string{chanGroup},
			wantCommitCalls: 1,
		},
		{
			name:   "commit request shape",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "Empty"},
				}}, nil
			},
			wantSeeded:      []string{chanGroup},
			wantCommitCalls: 1,
			checkCommit: func(t *testing.T, calls []*kafka.OffsetCommitRequest) {
				req := calls[0]
				if req.GroupID != chanGroup {
					t.Errorf("GroupID = %q, want %q", req.GroupID, chanGroup)
				}
				if req.GenerationID != -1 {
					t.Errorf("GenerationID = %d, want -1", req.GenerationID)
				}
				if req.MemberID != "" {
					t.Errorf("MemberID = %q, want empty", req.MemberID)
				}
				wantTopics := map[string][]kafka.OffsetCommit{
					"a": {{Partition: 0, Offset: 5}},
					"b": {{Partition: 0, Offset: 0}},
				}
				if len(req.Topics) != len(wantTopics) {
					t.Fatalf("Topics = %+v, want %+v", req.Topics, wantTopics)
				}
				for topic, want := range wantTopics {
					got, ok := req.Topics[topic]
					if !ok {
						t.Fatalf("Topics missing %q", topic)
					}
					if len(got) != len(want) || got[0] != want[0] {
						t.Errorf("Topics[%q] = %+v, want %+v", topic, got, want)
					}
				}
			},
		},
		{
			name:   "Dead group is seedable",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "Dead"},
				}}, nil
			},
			wantSeeded:      []string{chanGroup},
			wantCommitCalls: 1,
		},
		{
			name:   "absent group is seedable",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{}}, nil
			},
			wantSeeded:      []string{chanGroup},
			wantCommitCalls: 1,
			wantStates:      map[string]string{chanGroup: ""},
		},
		{
			name:   "per-group describe error is seedable",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, Error: kafka.GroupIdNotFound},
				}}, nil
			},
			wantSeeded:      []string{chanGroup},
			wantCommitCalls: 1,
			wantStates:      map[string]string{chanGroup: ""},
		},
		{
			name:   "describe transport error is seedable",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return nil, errors.New("boom")
			},
			wantSeeded:      []string{chanGroup},
			wantCommitCalls: 1,
			wantStates:      map[string]string{chanGroup: ""},
		},
		{
			name:   "Stable group is skipped",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "Stable"},
				}}, nil
			},
			wantSkipped:     []string{chanGroup},
			wantCommitCalls: 0,
			wantStates:      map[string]string{chanGroup: "Stable"},
		},
		{
			name:   "future state is skipped",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "AssigningPartitions"},
				}}, nil
			},
			wantSkipped:     []string{chanGroup},
			wantCommitCalls: 0,
		},
		{
			name:   "commit race is a non-fatal skip",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "Empty"},
				}}, nil
			},
			commitFn: func(*kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error) {
				return &kafka.OffsetCommitResponse{Topics: map[string][]kafka.OffsetCommitPartition{
					"a": {{Partition: 0, Error: kafka.UnknownMemberId}},
				}}, nil
			},
			wantSkipped:     []string{chanGroup},
			wantSeeded:      nil,
			wantCommitCalls: 1,
		},
		{
			name:   "other commit error is fatal",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "Empty"},
				}}, nil
			},
			commitFn: func(*kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error) {
				return &kafka.OffsetCommitResponse{Topics: map[string][]kafka.OffsetCommitPartition{
					"a": {{Partition: 0, Error: kafka.OffsetMetadataTooLarge}},
				}}, nil
			},
			wantErr:         chanGroup,
			wantCommitCalls: 1,
		},
		{
			name:   "commit transport error is fatal",
			groups: []string{chanGroup},
			describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
				return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
					{GroupID: chanGroup, GroupState: "Empty"},
				}}, nil
			},
			commitFn: func(*kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error) {
				return nil, errors.New("boom")
			},
			wantErr:         chanGroup,
			wantCommitCalls: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubClient{describeFn: tc.describeFn, commitFn: tc.commitFn}
			result, err := Seed(context.Background(), stub, nil, tc.groups, partitions, offsets, noWaitRetry())

			if tc.wantErr != "" {
				mustContain(t, err, tc.wantErr)
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !equalStrings(result.Seeded, tc.wantSeeded) {
				t.Errorf("Seeded = %v, want %v", result.Seeded, tc.wantSeeded)
			}
			if !equalStrings(result.Skipped, tc.wantSkipped) {
				t.Errorf("Skipped = %v, want %v", result.Skipped, tc.wantSkipped)
			}
			for group, state := range tc.wantStates {
				if got := result.States[group]; got != state {
					t.Errorf("States[%q] = %q, want %q", group, got, state)
				}
			}
			if len(tc.groups) == 0 && len(stub.describeCalls) != 0 {
				t.Errorf("describeCalls = %d, want 0", len(stub.describeCalls))
			}
			if len(stub.commitCalls) != tc.wantCommitCalls {
				t.Errorf("commitCalls = %d, want %d", len(stub.commitCalls), tc.wantCommitCalls)
			}
			if tc.checkCommit != nil {
				tc.checkCommit(t, stub.commitCalls)
			}
		})
	}
}

func TestSeed_MixedGroups(t *testing.T) {
	partitions, offsets := sharedFixtures()
	worldGroup := "World Service [pr-1450]"

	stub := &stubClient{
		describeFn: func(req *kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
			id := req.GroupIDs[0]
			state := "Empty"
			if id == worldGroup {
				state = "Stable"
			}
			return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
				{GroupID: id, GroupState: state},
			}}, nil
		},
	}

	result, err := Seed(context.Background(), stub, nil, []string{worldGroup, chanGroup}, partitions, offsets, noWaitRetry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalStrings(result.Skipped, []string{worldGroup}) {
		t.Errorf("Skipped = %v, want [%s]", result.Skipped, worldGroup)
	}
	if !equalStrings(result.Seeded, []string{chanGroup}) {
		t.Errorf("Seeded = %v, want [%s]", result.Seeded, chanGroup)
	}
	if len(stub.commitCalls) != 1 {
		t.Fatalf("commitCalls = %d, want 1", len(stub.commitCalls))
	}
	if stub.commitCalls[0].GroupID != chanGroup {
		t.Errorf("commit GroupID = %q, want %q", stub.commitCalls[0].GroupID, chanGroup)
	}
}

func TestSeed_DescribeIsRetried(t *testing.T) {
	partitions, offsets := sharedFixtures()

	attempt := 0
	stub := &stubClient{
		describeFn: func(*kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
			attempt++
			if attempt <= 2 {
				return nil, kafka.NotCoordinatorForGroup
			}
			return &kafka.DescribeGroupsResponse{Groups: []kafka.DescribeGroupsResponseGroup{
				{GroupID: chanGroup, GroupState: "Empty"},
			}}, nil
		},
	}

	result, err := Seed(context.Background(), stub, nil, []string{chanGroup}, partitions, offsets, noWaitRetry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalStrings(result.Seeded, []string{chanGroup}) {
		t.Errorf("Seeded = %v, want [%s]", result.Seeded, chanGroup)
	}
	if len(stub.describeCalls) != 3 {
		t.Errorf("describeCalls = %d, want 3", len(stub.describeCalls))
	}
}

func TestVerify(t *testing.T) {
	partitions, _ := sharedFixtures()

	seededResult := SeedResult{Seeded: []string{chanGroup}}
	skippedResult := SeedResult{Skipped: []string{chanGroup}}

	cases := []struct {
		name          string
		groups        []string
		seeded        SeedResult
		fetchFn       func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error)
		wantErr       string
		wantReports   int
		wantFetchZero bool
		checkReports  func(t *testing.T, reports []VerifyReport)
	}{
		{
			name:          "empty group list",
			groups:        []string{},
			wantFetchZero: true,
		},
		{
			name:   "seeded group fully committed",
			groups: []string{chanGroup},
			seeded: seededResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
					"a": {{Partition: 0, CommittedOffset: 5}},
					"b": {{Partition: 0, CommittedOffset: 0}},
				}}, nil
			},
			wantReports: 1,
			checkReports: func(t *testing.T, reports []VerifyReport) {
				if len(reports[0].Missing) != 0 {
					t.Errorf("Missing = %v, want empty", reports[0].Missing)
				}
			},
		},
		{
			name:   "seeded group missing an offset",
			groups: []string{chanGroup},
			seeded: seededResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
					"a": {{Partition: 0, CommittedOffset: -1}},
					"b": {{Partition: 0, CommittedOffset: 0}},
				}}, nil
			},
			wantErr: "a",
		},
		{
			name:   "seeded group missing from response",
			groups: []string{chanGroup},
			seeded: seededResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
					"a": {{Partition: 0, CommittedOffset: 5}},
				}}, nil
			},
			wantErr: "b",
		},
		{
			name:   "skipped group missing offsets warns",
			groups: []string{chanGroup},
			seeded: skippedResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
					"a": {{Partition: 0, CommittedOffset: -1}},
					"b": {{Partition: 0, CommittedOffset: -1}},
				}}, nil
			},
			wantReports: 1,
			checkReports: func(t *testing.T, reports []VerifyReport) {
				if reports[0].Total != 2 {
					t.Errorf("Total = %d, want 2", reports[0].Total)
				}
				if !equalStrings(reports[0].Missing, []string{"a", "b"}) {
					t.Errorf("Missing = %v, want [a b]", reports[0].Missing)
				}
			},
		},
		{
			name:   "skipped group fully committed",
			groups: []string{chanGroup},
			seeded: skippedResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
					"a": {{Partition: 0, CommittedOffset: 5}},
					"b": {{Partition: 0, CommittedOffset: 0}},
				}}, nil
			},
			wantReports: 1,
			checkReports: func(t *testing.T, reports []VerifyReport) {
				if len(reports[0].Missing) != 0 {
					t.Errorf("Missing = %v, want empty", reports[0].Missing)
				}
			},
		},
		{
			name:   "partition-level fetch error on a seeded group",
			groups: []string{chanGroup},
			seeded: seededResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
					"a": {{Partition: 0, Error: kafka.NotLeaderForPartition}},
				}}, nil
			},
			wantErr: "a",
		},
		{
			name:   "top-level fetch error, seeded",
			groups: []string{chanGroup},
			seeded: seededResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Error: kafka.GroupAuthorizationFailed}, nil
			},
			wantErr: chanGroup,
		},
		{
			name:   "top-level fetch error, skipped",
			groups: []string{chanGroup},
			seeded: skippedResult,
			fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
				return &kafka.OffsetFetchResponse{Error: kafka.GroupAuthorizationFailed}, nil
			},
			wantErr: chanGroup,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubClient{fetchFn: tc.fetchFn}
			reports, err := Verify(context.Background(), stub, nil, tc.groups, partitions, tc.seeded, noWaitRetry())

			if tc.wantErr != "" {
				mustContain(t, err, tc.wantErr)
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tc.wantFetchZero && len(stub.fetchCalls) != 0 {
				t.Errorf("fetchCalls = %d, want 0", len(stub.fetchCalls))
			}
			if len(reports) != tc.wantReports {
				t.Errorf("reports = %d, want %d", len(reports), tc.wantReports)
			}
			if tc.checkReports != nil {
				tc.checkReports(t, reports)
			}
		})
	}
}

func TestVerify_MissingListIsSorted(t *testing.T) {
	partitions := map[string][]int{"z": {0}, "a": {0}, "m": {0}}
	skipped := SeedResult{Skipped: []string{chanGroup}}

	stub := &stubClient{
		fetchFn: func(*kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
			return &kafka.OffsetFetchResponse{Topics: map[string][]kafka.OffsetFetchPartition{
				"z": {{Partition: 0, CommittedOffset: -1}},
				"a": {{Partition: 0, CommittedOffset: -1}},
				"m": {{Partition: 0, CommittedOffset: -1}},
			}}, nil
		},
	}

	reports, err := Verify(context.Background(), stub, nil, []string{chanGroup}, partitions, skipped, noWaitRetry())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 1 {
		t.Fatalf("reports = %d, want 1", len(reports))
	}
	want := []string{"a", "m", "z"}
	if !equalStrings(reports[0].Missing, want) {
		t.Errorf("Missing = %v, want %v", reports[0].Missing, want)
	}
}

// equalStrings compares two string slices in exact order — Seeded, Skipped,
// and Missing are all contractually ordered (input order or sorted), so a
// set comparison here would hide an ordering bug.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
