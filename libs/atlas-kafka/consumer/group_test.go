package consumer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

// TestGroupConfigMirrorsTodaysTopology pins Option A: the new engine joins
// with the same group ID and the same single-topic subscription as the legacy
// reader, so broker-visible group topology is unchanged and rollback needs no
// group-state migration (FR-3.4, FR-5.2).
func TestGroupConfigMirrorsTodaysTopology(t *testing.T) {
	c := &Consumer{
		topic:       "EVENT_TOPIC_TEST",
		groupId:     "Test Service",
		brokers:     []string{"broker-a:9092", "broker-b:9092"},
		maxWait:     10 * time.Second,
		startOffset: kafka.FirstOffset,
	}

	cfg := c.groupConfig()

	if cfg.ID != "Test Service" {
		t.Fatalf("ID = %q, want %q", cfg.ID, "Test Service")
	}
	if len(cfg.Topics) != 1 || cfg.Topics[0] != "EVENT_TOPIC_TEST" {
		t.Fatalf("Topics = %v, want [EVENT_TOPIC_TEST]", cfg.Topics)
	}
	if len(cfg.Brokers) != 2 || cfg.Brokers[0] != "broker-a:9092" {
		t.Fatalf("Brokers = %v", cfg.Brokers)
	}
	if cfg.StartOffset != kafka.FirstOffset {
		t.Fatalf("StartOffset = %d, want FirstOffset (%d)", cfg.StartOffset, kafka.FirstOffset)
	}
	// Deliberate divergence from the legacy engine (task-267 FR-1.1/FR-1.2).
	// The `reader` engine inherits kafka-go's ReaderConfig default of false and
	// cannot be changed without touching the frozen rollback path; the
	// consumergroup engine enables the watch as one of THREE parts of the
	// task-267 fix. It is only safe doing so because awaitTopic and
	// classifyEmptyAssignment keep this member out of the group whenever the
	// topic is absent — on its own, the watch's startup read fails with
	// UnknownTopicOrPartition, ends the generation, and rejoins with no
	// backoff (design §2.1-2.3).
	if !cfg.WatchPartitionChanges {
		t.Fatal("WatchPartitionChanges = false, want true (task-267 FR-1.1)")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestGroupConfigHonoursLastOffset covers SetStartOffset(kafka.LastOffset).
func TestGroupConfigHonoursLastOffset(t *testing.T) {
	c := &Consumer{
		topic:       "t",
		groupId:     "g",
		brokers:     []string{"b:9092"},
		maxWait:     time.Second,
		startOffset: kafka.LastOffset,
	}
	if got := c.groupConfig().StartOffset; got != kafka.LastOffset {
		t.Fatalf("StartOffset = %d, want LastOffset (%d)", got, kafka.LastOffset)
	}
}

// TestPartitionReaderConfigHasNoGroupID: a partition reader is positioned by
// SetOffset, which kafka-go rejects on a group reader
// (errNotAvailableWithGroup, reader.go:36). Setting both Partition and GroupID
// also fails ReaderConfig.Validate (reader.go:545-547).
func TestPartitionReaderConfigHasNoGroupID(t *testing.T) {
	c := &Consumer{
		topic:       "EVENT_TOPIC_TEST",
		groupId:     "Test Service",
		brokers:     []string{"broker-a:9092"},
		maxWait:     7 * time.Second,
		startOffset: kafka.FirstOffset,
	}

	cfg := c.partitionReaderConfig(3)

	if cfg.GroupID != "" {
		t.Fatalf("GroupID = %q, want empty", cfg.GroupID)
	}
	if cfg.Partition != 3 {
		t.Fatalf("Partition = %d, want 3", cfg.Partition)
	}
	if cfg.Topic != "EVENT_TOPIC_TEST" {
		t.Fatalf("Topic = %q", cfg.Topic)
	}
	if cfg.MaxWait != 7*time.Second {
		t.Fatalf("MaxWait = %v, want 7s", cfg.MaxWait)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestManagerDefaultProducersArePresent: GetManager must install working
// defaults so a service that configures nothing still gets a real group and
// real partition readers.
func TestManagerDefaultProducersArePresent(t *testing.T) {
	ResetInstance()
	m := GetManager()
	if m.gp == nil {
		t.Fatal("Manager.gp is nil; GetManager must install a default GroupProducer")
	}
	if m.prp == nil {
		t.Fatal("Manager.prp is nil; GetManager must install a default PartitionReaderProducer")
	}
	if m.pcp == nil {
		t.Fatal("Manager.pcp is nil; GetManager must install a default PartitionCountProducer")
	}
	ResetInstance()
}

// TestConfigProducersOverrideDefaults covers the additive test seams.
func TestConfigProducersOverrideDefaults(t *testing.T) {
	ResetInstance()
	var gpCalled, prpCalled, pcpCalled bool
	m := GetManager(
		ConfigGroupProducer(func(kafka.ConsumerGroupConfig) (Group, error) { gpCalled = true; return nil, nil }),
		ConfigPartitionReaderProducer(func(kafka.ReaderConfig, int64) KafkaReader { prpCalled = true; return nil }),
		ConfigPartitionCountProducer(func(context.Context, []string, string) (int, error) { pcpCalled = true; return 1, nil }),
	)
	_, _ = m.gp(kafka.ConsumerGroupConfig{})
	_ = m.prp(kafka.ReaderConfig{}, 0)
	_, _ = m.pcp(context.Background(), nil, "t")
	if !gpCalled || !prpCalled || !pcpCalled {
		t.Fatalf("configured producers not used: gp=%v prp=%v pcp=%v", gpCalled, prpCalled, pcpCalled)
	}
	ResetInstance()
}

// TestPartitionCountFromMetadata pins the FR-2.5 boundary: only a DEFINITE
// negative maps to ErrTopicNotFound. Every other topic error stays itself, so
// the callers treat it as indeterminate and fall through to today's behaviour.
func TestPartitionCountFromMetadata(t *testing.T) {
	tests := []struct {
		name      string
		res       *kafka.MetadataResponse
		wantCount int
		wantErr   error
	}{
		{"two partitions", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Partitions: []kafka.Partition{{ID: 0}, {ID: 1}}}}}, 2, nil},
		{"one partition", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Partitions: []kafka.Partition{{ID: 0}}}}}, 1, nil},
		{"unknown topic", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Error: kafka.UnknownTopicOrPartition}}}, 0, ErrTopicNotFound},
		{"zero partitions", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t"}}}, 0, ErrTopicNotFound},
		{"topic absent", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "other", Partitions: []kafka.Partition{{ID: 0}}}}}, 0, ErrTopicNotFound},
		{"empty response", &kafka.MetadataResponse{}, 0, ErrTopicNotFound},
		{"other topic error is indeterminate", &kafka.MetadataResponse{Topics: []kafka.Topic{{Name: "t", Error: kafka.LeaderNotAvailable}}}, 0, kafka.LeaderNotAvailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := partitionCountFromMetadata("t", tt.res)
			if got != tt.wantCount {
				t.Fatalf("count = %d, want %d", got, tt.wantCount)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && tt.wantErr != ErrTopicNotFound && errors.Is(err, ErrTopicNotFound) {
				t.Fatalf("indeterminate error %v was collapsed into ErrTopicNotFound", err)
			}
		})
	}
}
