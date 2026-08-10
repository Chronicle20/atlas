package consumer

import (
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
	// Today's ReaderConfig never sets WatchPartitionChanges, and kafka-go
	// forwards that false into its internal ConsumerGroupConfig
	// (reader.go:731-732). Enabling it here would be a behaviour change
	// smuggled into a migration (PRD §9.5).
	if cfg.WatchPartitionChanges {
		t.Fatal("WatchPartitionChanges = true, want false to match the legacy engine")
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
	ResetInstance()
}

// TestConfigProducersOverrideDefaults covers the additive test seams.
func TestConfigProducersOverrideDefaults(t *testing.T) {
	ResetInstance()
	var gpCalled, prpCalled bool
	m := GetManager(
		ConfigGroupProducer(func(kafka.ConsumerGroupConfig) (Group, error) { gpCalled = true; return nil, nil }),
		ConfigPartitionReaderProducer(func(kafka.ReaderConfig, int64) KafkaReader { prpCalled = true; return nil }),
	)
	_, _ = m.gp(kafka.ConsumerGroupConfig{})
	_ = m.prp(kafka.ReaderConfig{}, 0)
	if !gpCalled || !prpCalled {
		t.Fatalf("configured producers not used: gp=%v prp=%v", gpCalled, prpCalled)
	}
	ResetInstance()
}
