package consumer

import (
	"context"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
)

// Group is the subset of *kafka.ConsumerGroup this package depends on.
// Defining it as an interface is what lets the assignment-awareness tests
// script generations without a broker (task-209 design §6).
type Group interface {
	// Next blocks until the previous generation has ended and the next one
	// is ready. It returns ErrGroupClosed after Close, or the ctx error.
	Next(ctx context.Context) (Generation, error)
	Close() error
}

// Generation is the subset of *kafka.Generation this package depends on.
//
// IMPORTANT: a function passed to Start ENDS THE GENERATION when it returns
// (kafka-go consumergroup.go:387-405). Every recoverable error must therefore
// be handled inside the function; returning is reserved for "the generation
// is already ending". That inversion — recover in place, never by rejoining —
// is the point of the migration (design §1.1).
type Generation interface {
	ID() int32
	Assignments() map[string][]kafka.PartitionAssignment
	Start(fn func(ctx context.Context))
	CommitOffsets(offsets map[string]map[int]int64) error
}

// GroupProducer builds a Group from a kafka-go consumer-group config.
type GroupProducer func(cfg kafka.ConsumerGroupConfig) (Group, error)

// PartitionReaderProducer builds a positioned, non-group reader for a single
// partition. The offset is a producer argument rather than a KafkaReader
// method so the existing KafkaReader seam stays frozen (FR-3.5): every mock
// written against KafkaReader works unchanged against the new fetch loop.
type PartitionReaderProducer func(cfg kafka.ReaderConfig, offset int64) KafkaReader

//goland:noinspection GoUnusedExportedFunction
func ConfigGroupProducer(gp GroupProducer) ManagerConfig {
	return func(m *Manager) {
		m.gp = gp
	}
}

//goland:noinspection GoUnusedExportedFunction
func ConfigPartitionReaderProducer(prp PartitionReaderProducer) ManagerConfig {
	return func(m *Manager) {
		m.prp = prp
	}
}

// ErrTopicNotFound reports that the broker knows no partitions for the topic:
// either UNKNOWN_TOPIC_OR_PARTITION, or a successful metadata response whose
// topic entry carries zero partitions. It is a DEFINITE negative — every other
// error is indeterminate (FR-2.5), because acting on a broker blip by leaving
// a group is worse than the deafness this task fixes.
var ErrTopicNotFound = errors.New("topic not found or has no partitions")

// topicMetadataTimeout bounds a single partition-count lookup, and doubles as
// the metadata client's own Timeout. 5s is the same order as kafka-go's
// default PartitionWatchInterval; the lookup is off the message path, so the
// worst case is a delayed join, never a stalled fetch.
const topicMetadataTimeout = 5 * time.Second

// PartitionCountProducer reports the current partition count for a topic.
// Shaped as a producer function, like GroupProducer and
// PartitionReaderProducer, so both the pre-join gate and the empty-assignment
// classification can be scripted without a broker (FR-2.2). Group stays a pure
// subset of *kafka.ConsumerGroup: the gate needs this lookup BEFORE a Group
// exists at all.
type PartitionCountProducer func(ctx context.Context, brokers []string, topic string) (int, error)

//goland:noinspection GoUnusedExportedFunction
func ConfigPartitionCountProducer(pcp PartitionCountProducer) ManagerConfig {
	return func(m *Manager) {
		m.pcp = pcp
	}
}

// defaultPartitionCountProducer asks the cluster for one topic's metadata.
//
// kafka.Client.Metadata, NOT kafka.Conn.ReadPartitions: ReadPartitions sends
// topicMetadataRequestV6{... AllowAutoTopicCreation: true} (kafka-go
// conn.go:984-986), and a consumer must never create a topic as a side effect
// of asking whether it exists. Client.Metadata builds metadataAPI.Request with
// TopicNames only (kafka-go metadata.go:40-44) and returns the per-topic error
// as a structured field rather than collapsing the response into one error —
// which is what lets partitionCountFromMetadata separate "no such topic" from
// "broker unreachable".
func defaultPartitionCountProducer(ctx context.Context, brokers []string, topic string) (int, error) {
	client := &kafka.Client{Addr: kafka.TCP(brokers...), Timeout: topicMetadataTimeout}
	res, err := client.Metadata(ctx, &kafka.MetadataRequest{Topics: []string{topic}})
	if err != nil {
		return 0, err
	}
	return partitionCountFromMetadata(topic, res)
}

// partitionCountFromMetadata is the pure mapping half of
// defaultPartitionCountProducer, split out so the FR-2.5 boundary is testable
// without a broker.
func partitionCountFromMetadata(topic string, res *kafka.MetadataResponse) (int, error) {
	if res == nil {
		return 0, ErrTopicNotFound
	}
	for _, t := range res.Topics {
		if t.Name != topic {
			continue
		}
		if t.Error != nil {
			if errors.Is(t.Error, kafka.UnknownTopicOrPartition) {
				return 0, ErrTopicNotFound
			}
			return 0, t.Error
		}
		if len(t.Partitions) == 0 {
			return 0, ErrTopicNotFound
		}
		return len(t.Partitions), nil
	}
	return 0, ErrTopicNotFound
}

// defaultGroupProducer wraps kafka.NewConsumerGroup.
func defaultGroupProducer(cfg kafka.ConsumerGroupConfig) (Group, error) {
	cg, err := kafka.NewConsumerGroup(cfg)
	if err != nil {
		return nil, err
	}
	return &kafkaGroup{cg: cg}, nil
}

// defaultPartitionReaderProducer builds a plain (non-group) reader and
// positions it. SetOffset only fails on a closed reader or a group reader
// (kafka-go reader.go SetOffset); this reader is neither, and a no-op when
// offset already equals the reader's default FirstOffset.
func defaultPartitionReaderProducer(cfg kafka.ReaderConfig, offset int64) KafkaReader {
	r := kafka.NewReader(cfg)
	_ = r.SetOffset(offset)
	return r
}

type kafkaGroup struct {
	cg *kafka.ConsumerGroup
}

func (g *kafkaGroup) Next(ctx context.Context) (Generation, error) {
	gen, err := g.cg.Next(ctx)
	if err != nil {
		return nil, err
	}
	return &kafkaGeneration{gen: gen}, nil
}

func (g *kafkaGroup) Close() error { return g.cg.Close() }

type kafkaGeneration struct {
	gen *kafka.Generation
}

func (g *kafkaGeneration) ID() int32 { return g.gen.ID }

func (g *kafkaGeneration) Assignments() map[string][]kafka.PartitionAssignment {
	return g.gen.Assignments
}

func (g *kafkaGeneration) Start(fn func(ctx context.Context)) { g.gen.Start(fn) }

func (g *kafkaGeneration) CommitOffsets(offsets map[string]map[int]int64) error {
	return g.gen.CommitOffsets(offsets)
}

// groupConfig builds the consumer-group config for this Consumer. It is a
// deliberate 1:1 mirror of what kafka-go derives internally from today's
// ReaderConfig (reader.go:717-733): same group ID, same single-topic
// subscription, same StartOffset, WatchPartitionChanges left false.
func (c *Consumer) groupConfig() kafka.ConsumerGroupConfig {
	return kafka.ConsumerGroupConfig{
		ID:                    c.groupId,
		Brokers:               append([]string(nil), c.brokers...),
		Topics:                []string{c.topic},
		StartOffset:           c.startOffset,
		WatchPartitionChanges: false,
	}
}

// partitionReaderConfig builds the config for one assigned partition's
// reader. GroupID is deliberately absent: group membership belongs to the
// ConsumerGroup, and kafka-go rejects a reader with both Partition and
// GroupID set (reader.go:545-547).
func (c *Consumer) partitionReaderConfig(partition int) kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers:   append([]string(nil), c.brokers...),
		Topic:     c.topic,
		Partition: partition,
		MaxWait:   c.maxWait,
	}
}
