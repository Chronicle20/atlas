package topics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"atlas.com/kafka-precreate/internal/discover"
	"atlas.com/kafka-precreate/internal/kafkaops"
)

// stubClient implements kafkaops.AdminClient for tests in this package. The
// four request/response methods this package actually calls record every
// call and delegate to their *Fn field, returning a zero-value response when
// unset. The three group-coordinator methods panic: topics must never issue
// them, and the panic is the proof.
type stubClient struct {
	createCalls []*kafka.CreateTopicsRequest
	createFn    func(*kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error)

	alterCalls []*kafka.IncrementalAlterConfigsRequest
	alterFn    func(*kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error)

	metadataCalls []*kafka.MetadataRequest
	metadataFn    func(*kafka.MetadataRequest) (*kafka.MetadataResponse, error)

	listCalls []*kafka.ListOffsetsRequest
	listFn    func(*kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error)
}

func (s *stubClient) CreateTopics(_ context.Context, req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
	s.createCalls = append(s.createCalls, req)
	if s.createFn == nil {
		return &kafka.CreateTopicsResponse{}, nil
	}
	return s.createFn(req)
}

func (s *stubClient) IncrementalAlterConfigs(_ context.Context, req *kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error) {
	s.alterCalls = append(s.alterCalls, req)
	if s.alterFn == nil {
		return &kafka.IncrementalAlterConfigsResponse{}, nil
	}
	return s.alterFn(req)
}

func (s *stubClient) Metadata(_ context.Context, req *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
	s.metadataCalls = append(s.metadataCalls, req)
	if s.metadataFn == nil {
		return &kafka.MetadataResponse{}, nil
	}
	return s.metadataFn(req)
}

func (s *stubClient) ListOffsets(_ context.Context, req *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
	s.listCalls = append(s.listCalls, req)
	if s.listFn == nil {
		return &kafka.ListOffsetsResponse{}, nil
	}
	return s.listFn(req)
}

func (s *stubClient) DescribeGroups(context.Context, *kafka.DescribeGroupsRequest) (*kafka.DescribeGroupsResponse, error) {
	panic("unexpected call")
}

func (s *stubClient) OffsetCommit(context.Context, *kafka.OffsetCommitRequest) (*kafka.OffsetCommitResponse, error) {
	panic("unexpected call")
}

func (s *stubClient) OffsetFetch(context.Context, *kafka.OffsetFetchRequest) (*kafka.OffsetFetchResponse, error) {
	panic("unexpected call")
}

// fakeClock provides a deterministic Sleep/Now pair for Settle tests. No
// test in this file sleeps for real. This duplicates kafkaops' fakeClock
// deliberately: it is test-local, package-local, and small enough that
// sharing it would cost more than it saves.
type fakeClock struct {
	now   time.Time
	slept []time.Duration
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(0, 0)}
}

func (c *fakeClock) Sleep(d time.Duration) {
	c.now = c.now.Add(d)
	c.slept = append(c.slept, d)
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func durationsEqual(a, b []time.Duration) bool {
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

var testAddr net.Addr = &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9092}

func plainNames(prefix string, n int) []string {
	names := make([]string, n)
	for i := 0; i < n; i++ {
		names[i] = fmt.Sprintf("%s-%03d", prefix, i)
	}
	return names
}

func TestEnsure_SingleCreateRequest(t *testing.T) {
	plain := plainNames("plain", 170)
	compact := []string{"cfg-env", "cfg-service", "cfg-tenant"}
	topicsIn := discover.Topics{Plain: plain, Compact: compact}

	errs := make(map[string]error, 173)
	for _, name := range plain {
		errs[name] = nil
	}
	for _, name := range compact {
		errs[name] = nil
	}

	stub := &stubClient{
		createFn: func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
			return &kafka.CreateTopicsResponse{Errors: errs}, nil
		},
	}

	result, err := Ensure(context.Background(), stub, testAddr, topicsIn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(stub.createCalls))
	}
	req := stub.createCalls[0]
	if len(req.Topics) != 173 {
		t.Fatalf("expected 173 topic configs, got %d", len(req.Topics))
	}

	compactSet := map[string]struct{}{"cfg-env": {}, "cfg-service": {}, "cfg-tenant": {}}
	for _, cfg := range req.Topics {
		if cfg.NumPartitions != 1 {
			t.Errorf("topic %s: expected NumPartitions 1, got %d", cfg.Topic, cfg.NumPartitions)
		}
		if cfg.ReplicationFactor != 1 {
			t.Errorf("topic %s: expected ReplicationFactor 1, got %d", cfg.Topic, cfg.ReplicationFactor)
		}
		if _, isCompact := compactSet[cfg.Topic]; isCompact {
			want := []kafka.ConfigEntry{
				{ConfigName: "cleanup.policy", ConfigValue: "compact"},
				{ConfigName: "max.compaction.lag.ms", ConfigValue: "600000"},
				{ConfigName: "segment.ms", ConfigValue: "600000"},
				{ConfigName: "min.cleanable.dirty.ratio", ConfigValue: "0.01"},
			}
			if len(cfg.ConfigEntries) != len(want) {
				t.Fatalf("topic %s: expected %d ConfigEntries, got %d", cfg.Topic, len(want), len(cfg.ConfigEntries))
			}
			for i, w := range want {
				if cfg.ConfigEntries[i] != w {
					t.Errorf("topic %s entry %d: expected %+v, got %+v", cfg.Topic, i, w, cfg.ConfigEntries[i])
				}
			}
		} else if len(cfg.ConfigEntries) != 0 {
			t.Errorf("topic %s: expected no ConfigEntries, got %v", cfg.Topic, cfg.ConfigEntries)
		}
	}

	if result != (EnsureResult{Created: 173, Existing: 0}) {
		t.Errorf("expected EnsureResult{173, 0}, got %+v", result)
	}
}

func TestEnsure_CreateErrors(t *testing.T) {
	transportErr := errors.New("dial tcp: connection refused")

	tests := []struct {
		name         string
		errs         map[string]error
		useCreateFn  func(*kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error)
		expectErr    bool
		expectErrIs  []error
		expectNames  []string
		expectResult EnsureResult
		noAlterCall  bool
	}{
		{
			name:         "all created",
			errs:         map[string]error{"a": nil, "b": nil, "c": nil},
			expectResult: EnsureResult{Created: 3, Existing: 0},
		},
		{
			name:         "all already exist",
			errs:         map[string]error{"a": kafka.TopicAlreadyExists, "b": kafka.TopicAlreadyExists, "c": kafka.TopicAlreadyExists},
			expectResult: EnsureResult{Created: 0, Existing: 3},
		},
		{
			name:         "mixed",
			errs:         map[string]error{"a": nil, "b": kafka.TopicAlreadyExists, "c": nil},
			expectResult: EnsureResult{Created: 2, Existing: 1},
		},
		{
			name:        "one fatal",
			errs:        map[string]error{"a": nil, "b": kafka.InvalidTopic, "c": nil},
			expectErr:   true,
			expectErrIs: []error{kafka.InvalidTopic},
			expectNames: []string{"b"},
		},
		{
			name:        "two fatal, both named",
			errs:        map[string]error{"a": kafka.InvalidTopic, "b": nil, "c": kafka.InvalidPartitionNumber},
			expectErr:   true,
			expectErrIs: []error{kafka.InvalidTopic, kafka.InvalidPartitionNumber},
			expectNames: []string{"a", "c"},
		},
		{
			name: "transport error",
			useCreateFn: func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
				return nil, transportErr
			},
			expectErr:   true,
			expectErrIs: []error{transportErr},
			noAlterCall: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stub := &stubClient{}
			if tc.useCreateFn != nil {
				stub.createFn = tc.useCreateFn
			} else {
				errs := tc.errs
				stub.createFn = func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
					return &kafka.CreateTopicsResponse{Errors: errs}, nil
				}
			}

			topicsIn := discover.Topics{Plain: []string{"a", "b"}, Compact: []string{"c"}}
			result, err := Ensure(context.Background(), stub, testAddr, topicsIn)

			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				for _, want := range tc.expectErrIs {
					if !errors.Is(err, want) {
						t.Errorf("expected errors.Is(err, %v) to be true, got %v", want, err)
					}
				}
				for _, name := range tc.expectNames {
					if !containsSubstring(err.Error(), name) {
						t.Errorf("expected error message %q to contain %q", err.Error(), name)
					}
				}
				if tc.noAlterCall && len(stub.alterCalls) != 0 {
					t.Errorf("expected no alter call, got %d", len(stub.alterCalls))
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tc.expectResult {
				t.Errorf("expected %+v, got %+v", tc.expectResult, result)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestEnsure_AlterConfigs(t *testing.T) {
	stub := &stubClient{
		createFn: func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
			return &kafka.CreateTopicsResponse{Errors: map[string]error{"p1": nil, "c1": nil, "c2": nil}}, nil
		},
	}

	topicsIn := discover.Topics{Plain: []string{"p1"}, Compact: []string{"c1", "c2"}}
	_, err := Ensure(context.Background(), stub, testAddr, topicsIn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.alterCalls) != 1 {
		t.Fatalf("expected 1 alter call, got %d", len(stub.alterCalls))
	}
	req := stub.alterCalls[0]
	if len(req.Resources) != 2 {
		t.Fatalf("expected 2 alter resources, got %d", len(req.Resources))
	}

	sorted := make([]kafka.IncrementalAlterConfigsRequestResource, len(req.Resources))
	copy(sorted, req.Resources)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ResourceName < sorted[j].ResourceName })

	wantNames := []string{"c1", "c2"}
	for i, res := range sorted {
		if res.ResourceName != wantNames[i] {
			t.Errorf("expected resource %d name %q, got %q", i, wantNames[i], res.ResourceName)
		}
		if res.ResourceType != kafka.ResourceTypeTopic {
			t.Errorf("expected ResourceType kafka.ResourceTypeTopic, got %v", res.ResourceType)
		}
		wantConfigs := []kafka.IncrementalAlterConfigsRequestConfig{
			{Name: "cleanup.policy", Value: "compact", ConfigOperation: kafka.ConfigOperationSet},
			{Name: "max.compaction.lag.ms", Value: "600000", ConfigOperation: kafka.ConfigOperationSet},
			{Name: "segment.ms", Value: "600000", ConfigOperation: kafka.ConfigOperationSet},
			{Name: "min.cleanable.dirty.ratio", Value: "0.01", ConfigOperation: kafka.ConfigOperationSet},
		}
		if len(res.Configs) != len(wantConfigs) {
			t.Fatalf("resource %q: expected %d configs, got %d", res.ResourceName, len(wantConfigs), len(res.Configs))
		}
		for j, want := range wantConfigs {
			if res.Configs[j] != want {
				t.Errorf("resource %q config %d: expected %+v, got %+v", res.ResourceName, j, want, res.Configs[j])
			}
		}
	}
}

func TestEnsure_AlterConfigs_NoCompactTopics(t *testing.T) {
	stub := &stubClient{
		createFn: func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
			return &kafka.CreateTopicsResponse{Errors: map[string]error{"p1": nil}}, nil
		},
	}

	topicsIn := discover.Topics{Plain: []string{"p1"}, Compact: []string{}}
	_, err := Ensure(context.Background(), stub, testAddr, topicsIn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stub.alterCalls) != 0 {
		t.Errorf("expected no alter call, got %d", len(stub.alterCalls))
	}
}

func TestEnsure_AlterConfigs_ResourceError(t *testing.T) {
	stub := &stubClient{
		createFn: func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
			return &kafka.CreateTopicsResponse{Errors: map[string]error{"c1": nil, "c2": nil}}, nil
		},
		alterFn: func(req *kafka.IncrementalAlterConfigsRequest) (*kafka.IncrementalAlterConfigsResponse, error) {
			return &kafka.IncrementalAlterConfigsResponse{
				Resources: []kafka.IncrementalAlterConfigsResponseResource{
					{ResourceName: "c1", Error: nil},
					{ResourceName: "c2", Error: kafka.PolicyViolation},
				},
			}, nil
		},
	}

	topicsIn := discover.Topics{Plain: []string{}, Compact: []string{"c1", "c2"}}
	_, err := Ensure(context.Background(), stub, testAddr, topicsIn)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !containsSubstring(err.Error(), "c2") {
		t.Errorf("expected error message %q to contain %q", err.Error(), "c2")
	}
}

// TestEnsure_CompactConfigsMatchAcrossRequests guards the defect class: a
// config present in the create-topics builder but absent from the
// alter-configs builder (or vice versa). It asserts both request bodies
// carry the same set of name/value pairs for every compacted topic, using
// literal values rather than the package constants so a change to the
// constants cannot silently pass.
func TestEnsure_CompactConfigsMatchAcrossRequests(t *testing.T) {
	stub := &stubClient{
		createFn: func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
			return &kafka.CreateTopicsResponse{Errors: map[string]error{"p1": nil, "c1": nil, "c2": nil}}, nil
		},
	}

	topicsIn := discover.Topics{Plain: []string{"p1"}, Compact: []string{"c1", "c2"}}
	_, err := Ensure(context.Background(), stub, testAddr, topicsIn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantPairs := map[string]string{
		"cleanup.policy":            "compact",
		"max.compaction.lag.ms":     "600000",
		"segment.ms":                "600000",
		"min.cleanable.dirty.ratio": "0.01",
	}

	if len(stub.createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(stub.createCalls))
	}
	compactCreateNames := map[string]struct{}{"c1": {}, "c2": {}}
	createMatched := 0
	for _, cfg := range stub.createCalls[0].Topics {
		if _, ok := compactCreateNames[cfg.Topic]; !ok {
			continue
		}
		createMatched++
		got := make(map[string]string, len(cfg.ConfigEntries))
		for _, e := range cfg.ConfigEntries {
			if _, dup := got[e.ConfigName]; dup {
				t.Fatalf("topic %q: duplicate ConfigName %q in create entries", cfg.Topic, e.ConfigName)
			}
			got[e.ConfigName] = e.ConfigValue
		}
		if !reflect.DeepEqual(got, wantPairs) {
			t.Errorf("topic %q: create ConfigEntries = %v, want %v", cfg.Topic, got, wantPairs)
		}
	}
	if createMatched == 0 {
		t.Fatalf("no compacted topics matched in create request; vacuous pass")
	}

	if len(stub.alterCalls) != 1 {
		t.Fatalf("expected 1 alter call, got %d", len(stub.alterCalls))
	}
	alterMatched := 0
	for _, res := range stub.alterCalls[0].Resources {
		alterMatched++
		got := make(map[string]string, len(res.Configs))
		for _, c := range res.Configs {
			if _, dup := got[c.Name]; dup {
				t.Fatalf("resource %q: duplicate Name %q in alter configs", res.ResourceName, c.Name)
			}
			got[c.Name] = c.Value
			if c.ConfigOperation != kafka.ConfigOperationSet {
				t.Errorf("resource %q config %q: expected ConfigOperationSet, got %v", res.ResourceName, c.Name, c.ConfigOperation)
			}
		}
		if !reflect.DeepEqual(got, wantPairs) {
			t.Errorf("resource %q: alter Configs = %v, want %v", res.ResourceName, got, wantPairs)
		}
	}
	if alterMatched == 0 {
		t.Fatalf("no resources matched in alter request; vacuous pass")
	}
}

// TestEnsure_PlainTopicsCarryNoConfig guards the plain-topic-not-configured
// half of the defect class as a standalone assertion, independent of any
// loop over compacted resources.
func TestEnsure_PlainTopicsCarryNoConfig(t *testing.T) {
	stub := &stubClient{
		createFn: func(req *kafka.CreateTopicsRequest) (*kafka.CreateTopicsResponse, error) {
			return &kafka.CreateTopicsResponse{Errors: map[string]error{"p1": nil, "p2": nil, "c1": nil}}, nil
		},
	}

	topicsIn := discover.Topics{Plain: []string{"p1", "p2"}, Compact: []string{"c1"}}
	_, err := Ensure(context.Background(), stub, testAddr, topicsIn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stub.createCalls) != 1 {
		t.Fatalf("expected 1 create call, got %d", len(stub.createCalls))
	}
	for _, name := range []string{"p1", "p2"} {
		found := false
		for _, cfg := range stub.createCalls[0].Topics {
			if cfg.Topic != name {
				continue
			}
			found = true
			if len(cfg.ConfigEntries) != 0 {
				t.Errorf("topic %q: expected no ConfigEntries, got %v", name, cfg.ConfigEntries)
			}
		}
		if !found {
			t.Fatalf("expected create request to contain topic %q", name)
		}
	}

	if len(stub.alterCalls) != 1 {
		t.Fatalf("expected 1 alter call, got %d", len(stub.alterCalls))
	}
	req := stub.alterCalls[0]
	if len(req.Resources) != 1 {
		t.Fatalf("expected 1 alter resource, got %d", len(req.Resources))
	}
	if req.Resources[0].ResourceName != "c1" {
		t.Errorf("expected alter resource name %q, got %q", "c1", req.Resources[0].ResourceName)
	}

	plainNames := map[string]struct{}{"p1": {}, "p2": {}}
	for _, res := range req.Resources {
		if _, isPlain := plainNames[res.ResourceName]; isPlain {
			t.Errorf("expected plain topic %q to not appear in alter resources", res.ResourceName)
		}
	}
}

func TestCompactConfigNames(t *testing.T) {
	want := []string{"cleanup.policy", "max.compaction.lag.ms", "segment.ms", "min.cleanable.dirty.ratio"}
	got := CompactConfigNames()
	if len(got) != len(want) {
		t.Fatalf("expected %d names, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func metadataResponse(topics ...kafka.Topic) *kafka.MetadataResponse {
	return &kafka.MetadataResponse{Topics: topics}
}

func partitions(topic string, ids ...int) kafka.Topic {
	parts := make([]kafka.Partition, len(ids))
	for i, id := range ids {
		parts[i] = kafka.Partition{Topic: topic, ID: id}
	}
	return kafka.Topic{Name: topic, Partitions: parts}
}

func TestSettle(t *testing.T) {
	transportErr := errors.New("boom")

	t.Run("present on first poll", func(t *testing.T) {
		clock := newFakeClock()
		stub := &stubClient{
			metadataFn: func(req *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
				return metadataResponse(partitions("a", 0), partitions("b", 0, 1)), nil
			},
		}
		cfg := SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		got, err := Settle(context.Background(), stub, testAddr, []string{"a", "b"}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string][]int{"a": {0}, "b": {0, 1}}
		if !partitionMapEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
		if len(stub.metadataCalls) != 1 {
			t.Errorf("expected 1 metadata call, got %d", len(stub.metadataCalls))
		}
		if len(clock.slept) != 0 {
			t.Errorf("expected no sleep, got %v", clock.slept)
		}
	})

	t.Run("appears on third poll", func(t *testing.T) {
		clock := newFakeClock()
		responses := []*kafka.MetadataResponse{
			metadataResponse(partitions("a", 0)),
			metadataResponse(partitions("a", 0)),
			metadataResponse(partitions("a", 0), partitions("b", 0)),
		}
		call := 0
		stub := &stubClient{
			metadataFn: func(req *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
				resp := responses[call]
				call++
				return resp, nil
			},
		}
		cfg := SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		got, err := Settle(context.Background(), stub, testAddr, []string{"a", "b"}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string][]int{"a": {0}, "b": {0}}
		if !partitionMapEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
		if len(stub.metadataCalls) != 3 {
			t.Errorf("expected 3 metadata calls, got %d", len(stub.metadataCalls))
		}
		wantBackoff := []time.Duration{250 * time.Millisecond, 250 * time.Millisecond}
		if !durationsEqual(clock.slept, wantBackoff) {
			t.Errorf("expected backoffs %v, got %v", wantBackoff, clock.slept)
		}
	})

	t.Run("topic present but zero partitions", func(t *testing.T) {
		clock := newFakeClock()
		responses := []*kafka.MetadataResponse{
			metadataResponse(kafka.Topic{Name: "a", Partitions: []kafka.Partition{}}),
			metadataResponse(partitions("a", 0)),
		}
		call := 0
		stub := &stubClient{
			metadataFn: func(req *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
				resp := responses[call]
				call++
				return resp, nil
			},
		}
		cfg := SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		got, err := Settle(context.Background(), stub, testAddr, []string{"a"}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string][]int{"a": {0}}
		if !partitionMapEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("topic carries a metadata error", func(t *testing.T) {
		clock := newFakeClock()
		responses := []*kafka.MetadataResponse{
			metadataResponse(kafka.Topic{Name: "a", Error: kafka.UnknownTopicOrPartition}),
			metadataResponse(partitions("a", 0)),
		}
		call := 0
		stub := &stubClient{
			metadataFn: func(req *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
				resp := responses[call]
				call++
				return resp, nil
			},
		}
		cfg := SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		got, err := Settle(context.Background(), stub, testAddr, []string{"a"}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string][]int{"a": {0}}
		if !partitionMapEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
	})

	t.Run("never appears", func(t *testing.T) {
		clock := newFakeClock()
		start := clock.now
		stub := &stubClient{
			metadataFn: func(req *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
				return metadataResponse(partitions("a", 0)), nil
			},
		}
		cfg := SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		_, err := Settle(context.Background(), stub, testAddr, []string{"a", "b"}, cfg)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !containsSubstring(err.Error(), "b") {
			t.Errorf("expected error to name %q, got %q", "b", err.Error())
		}
		if containsSubstring(err.Error(), "\"a\"") {
			t.Errorf("expected error to not name %q, got %q", "a", err.Error())
		}
		if clock.now.Sub(start) > 30*time.Second {
			t.Errorf("expected elapsed <= 30s, got %s", clock.now.Sub(start))
		}
	})

	t.Run("transport error", func(t *testing.T) {
		clock := newFakeClock()
		stub := &stubClient{
			metadataFn: func(req *kafka.MetadataRequest) (*kafka.MetadataResponse, error) {
				return nil, transportErr
			},
		}
		cfg := SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		_, err := Settle(context.Background(), stub, testAddr, []string{"a"}, cfg)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, transportErr) {
			t.Errorf("expected errors.Is(err, transportErr), got %v", err)
		}
	})

	t.Run("empty names", func(t *testing.T) {
		clock := newFakeClock()
		stub := &stubClient{}
		cfg := SettleConfig{Poll: 250 * time.Millisecond, Ceiling: 30 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		got, err := Settle(context.Background(), stub, testAddr, []string{}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
		if len(stub.metadataCalls) != 0 {
			t.Errorf("expected 0 metadata calls, got %d", len(stub.metadataCalls))
		}
	})
}

func partitionMapEqual(a, b map[string][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for topic, aParts := range a {
		bParts, ok := b[topic]
		if !ok || len(aParts) != len(bParts) {
			return false
		}
		for i := range aParts {
			if aParts[i] != bParts[i] {
				return false
			}
		}
	}
	return true
}

func TestEndOffsets(t *testing.T) {
	transportErr := errors.New("boom")

	t.Run("two topics", func(t *testing.T) {
		stub := &stubClient{
			listFn: func(req *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
				return &kafka.ListOffsetsResponse{
					Topics: map[string][]kafka.PartitionOffsets{
						"a": {{Partition: 0, LastOffset: 5}},
						"b": {{Partition: 0, LastOffset: 0}, {Partition: 1, LastOffset: 42}},
					},
				}, nil
			},
		}
		in := map[string][]int{"a": {0}, "b": {0, 1}}
		got, err := EndOffsets(context.Background(), stub, testAddr, in, kafkaops.RetryConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]map[int]int64{"a": {0: 5}, "b": {0: 0, 1: 42}}
		if !endOffsetsEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
		if len(stub.listCalls) != 1 {
			t.Fatalf("expected 1 ListOffsets call, got %d", len(stub.listCalls))
		}
		req := stub.listCalls[0]
		for topic, parts := range in {
			reqs, ok := req.Topics[topic]
			if !ok {
				t.Fatalf("expected request to carry topic %q", topic)
			}
			for _, p := range parts {
				found := false
				for _, r := range reqs {
					if r.Partition == p {
						found = true
						if r.Timestamp != kafka.LastOffset {
							t.Errorf("topic %s partition %d: expected Timestamp kafka.LastOffset, got %d", topic, p, r.Timestamp)
						}
					}
				}
				if !found {
					t.Errorf("expected request for topic %s partition %d", topic, p)
				}
			}
		}
	})

	t.Run("partition leader error retries then succeeds", func(t *testing.T) {
		// A partition can be visible in Metadata before it has an elected
		// leader; ListOffsets against it returns NotLeaderForPartition as a
		// per-partition error on an otherwise-successful response. The whole
		// request is idempotent, so it is re-issued under retry.
		clock := newFakeClock()
		responses := []*kafka.ListOffsetsResponse{
			{Topics: map[string][]kafka.PartitionOffsets{
				"a": {{Partition: 0, Error: kafka.NotLeaderForPartition}},
			}},
			{Topics: map[string][]kafka.PartitionOffsets{
				"a": {{Partition: 0, LastOffset: 5}},
			}},
		}
		call := 0
		stub := &stubClient{
			listFn: func(req *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
				resp := responses[call]
				call++
				return resp, nil
			},
		}
		cfg := kafkaops.RetryConfig{Base: 250 * time.Millisecond, Max: 2 * time.Second, Budget: 60 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		got, err := EndOffsets(context.Background(), stub, testAddr, map[string][]int{"a": {0}}, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := map[string]map[int]int64{"a": {0: 5}}
		if !endOffsetsEqual(got, want) {
			t.Errorf("expected %v, got %v", want, got)
		}
		if len(stub.listCalls) != 2 {
			t.Fatalf("expected 2 ListOffsets calls, got %d", len(stub.listCalls))
		}
		if len(clock.slept) != 1 {
			t.Errorf("expected 1 recorded backoff, got %v", clock.slept)
		}
	})

	t.Run("non-retriable partition error stays fatal on first call", func(t *testing.T) {
		clock := newFakeClock()
		stub := &stubClient{
			listFn: func(req *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
				return &kafka.ListOffsetsResponse{
					Topics: map[string][]kafka.PartitionOffsets{
						"a": {{Partition: 0, Error: kafka.UnknownTopicOrPartition}},
					},
				}, nil
			},
		}
		cfg := kafkaops.RetryConfig{Base: 250 * time.Millisecond, Max: 2 * time.Second, Budget: 60 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		_, err := EndOffsets(context.Background(), stub, testAddr, map[string][]int{"a": {0}}, cfg)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !containsSubstring(err.Error(), "a") || !containsSubstring(err.Error(), "0") {
			t.Errorf("expected error to name topic a and partition 0, got %q", err.Error())
		}
		if len(stub.listCalls) != 1 {
			t.Errorf("expected 1 ListOffsets call, got %d", len(stub.listCalls))
		}
	})

	t.Run("leader retry budget exhaustion names topic and partition", func(t *testing.T) {
		clock := newFakeClock()
		stub := &stubClient{
			listFn: func(req *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
				return &kafka.ListOffsetsResponse{
					Topics: map[string][]kafka.PartitionOffsets{
						"a": {{Partition: 0, Error: kafka.LeaderNotAvailable}},
					},
				}, nil
			},
		}
		cfg := kafkaops.RetryConfig{Base: 250 * time.Millisecond, Max: 2 * time.Second, Budget: 1 * time.Second, Sleep: clock.Sleep, Now: clock.Now}
		_, err := EndOffsets(context.Background(), stub, testAddr, map[string][]int{"a": {0}}, cfg)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !containsSubstring(err.Error(), "a") || !containsSubstring(err.Error(), "0") {
			t.Errorf("expected budget-exhaustion error to name topic a and partition 0, got %q", err.Error())
		}
		if !errors.Is(err, kafka.LeaderNotAvailable) {
			t.Errorf("expected errors.Is(err, kafka.LeaderNotAvailable), got %v", err)
		}
	})

	t.Run("missing partition in response", func(t *testing.T) {
		stub := &stubClient{
			listFn: func(req *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
				return &kafka.ListOffsetsResponse{
					Topics: map[string][]kafka.PartitionOffsets{
						"a": {{Partition: 0, LastOffset: 5}},
					},
				}, nil
			},
		}
		_, err := EndOffsets(context.Background(), stub, testAddr, map[string][]int{"a": {0, 1}}, kafkaops.RetryConfig{})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !containsSubstring(err.Error(), "a") || !containsSubstring(err.Error(), "1") {
			t.Errorf("expected error to name topic a and partition 1, got %q", err.Error())
		}
	})

	t.Run("transport error", func(t *testing.T) {
		stub := &stubClient{
			listFn: func(req *kafka.ListOffsetsRequest) (*kafka.ListOffsetsResponse, error) {
				return nil, transportErr
			},
		}
		_, err := EndOffsets(context.Background(), stub, testAddr, map[string][]int{"a": {0}}, kafkaops.RetryConfig{})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, transportErr) {
			t.Errorf("expected errors.Is(err, transportErr), got %v", err)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		stub := &stubClient{}
		got, err := EndOffsets(context.Background(), stub, testAddr, map[string][]int{}, kafkaops.RetryConfig{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected empty map, got %v", got)
		}
		if len(stub.listCalls) != 0 {
			t.Errorf("expected 0 ListOffsets calls, got %d", len(stub.listCalls))
		}
	})
}

func endOffsetsEqual(a, b map[string]map[int]int64) bool {
	if len(a) != len(b) {
		return false
	}
	for topic, aParts := range a {
		bParts, ok := b[topic]
		if !ok || len(aParts) != len(bParts) {
			return false
		}
		for p, off := range aParts {
			if bParts[p] != off {
				return false
			}
		}
	}
	return true
}
