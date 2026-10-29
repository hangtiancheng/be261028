package kafkax

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hangtiancheng/be261028/internal/infra/metrics"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

// EnsureTopics creates the given topics if they do not already exist. The local
// Kafka broker does not auto-create topics on produce, so the trade service calls
// this at startup to guarantee the team-success and team-refund topics exist.
// It uses a single partition and replication factor of 1, matching the local
// single-broker docker setup.
func EnsureTopics(brokers []string, clientID string, topics []string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("kafka brokers empty")
	}
	seed, err := kgo.NewClient(kgo.SeedBrokers(brokers...), kgo.ClientID(clientID+"-admin"))
	if err != nil {
		return fmt.Errorf("kafka admin client: %w", err)
	}
	defer seed.Close()

	adm := kadm.NewClient(seed)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := adm.CreateTopics(ctx, 1, 1, nil, topics...)
	if err != nil {
		return fmt.Errorf("create topics: %w", err)
	}
	for _, r := range resp {
		if r.Err != nil && !errors.Is(r.Err, kerr.TopicAlreadyExists) {
			return fmt.Errorf("create topic %s: %w", r.Topic, r.Err)
		}
	}
	return nil
}

// Producer publishes messages to Kafka topics, replacing the legacy RabbitMQ
// EventPublisher. Publish is synchronous so callers know the broker accepted the
// record before continuing (matching the old persistent convertAndSend semantics).
type Producer struct {
	client *kgo.Client
}

// NewProducer creates a synchronous Kafka producer.
func NewProducer(brokers []string, clientID string) (*Producer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers empty")
	}
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ProducerLinger(0),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.DisableIdempotentWrite(),
	}
	if clientID != "" {
		opts = append(opts, kgo.ClientID(clientID+"-producer"))
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}
	return &Producer{client: client}, nil
}

// Publish sends value to topic and blocks until the broker acknowledges it.
func (p *Producer) Publish(ctx context.Context, topic string, value []byte) error {
	rec := &kgo.Record{Topic: topic, Value: value, Timestamp: time.Now()}
	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		metrics.KafkaMessagesTotal.WithLabelValues(topic, "produce_error").Inc()
		return fmt.Errorf("kafka publish topic=%s: %w", topic, err)
	}
	metrics.KafkaMessagesTotal.WithLabelValues(topic, "produce").Inc()
	return nil
}

// Close flushes and closes the producer.
func (p *Producer) Close() {
	p.client.Close()
}

// Handler processes a single record.
type Handler func(ctx context.Context, topic string, value []byte) error

// OnError observes handler failures; the record is still committed to avoid
// poison-message loops (handlers are expected to be idempotent).
type OnError func(topic string, value []byte, err error)

// Consumer subscribes to topics within a consumer group and invokes a handler for
// each record, committing offsets after handling. It replaces the legacy RabbitMQ
// @RabbitListener consumers.
type Consumer struct {
	client  *kgo.Client
	topics  []string
	onError OnError
}

// NewConsumer creates a consumer-group consumer for the given topics.
func NewConsumer(brokers []string, groupID, clientID string, topics []string) (*Consumer, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers empty")
	}
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
	}
	if clientID != "" {
		opts = append(opts, kgo.ClientID(clientID+"-consumer"))
	}
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer: %w", err)
	}
	return &Consumer{client: client, topics: topics}, nil
}

// SetOnError registers a failure observer.
func (c *Consumer) SetOnError(fn OnError) { c.onError = fn }

// Run blocks polling for records until ctx is cancelled or the client is closed.
// Handler errors are reported via OnError but never terminate the loop; the batch
// is committed regardless so a malformed record cannot wedge the consumer.
func (c *Consumer) Run(ctx context.Context, handler Handler) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}
		iter := fetches.RecordIter()
		var batch []*kgo.Record
		for !iter.Done() {
			rec := iter.Next()
			metrics.KafkaMessagesTotal.WithLabelValues(rec.Topic, "consume").Inc()
			if err := handler(ctx, rec.Topic, rec.Value); err != nil {
				metrics.KafkaMessagesTotal.WithLabelValues(rec.Topic, "consume_error").Inc()
				if c.onError != nil {
					c.onError(rec.Topic, rec.Value, err)
				}
			}
			batch = append(batch, rec)
		}
		if len(batch) > 0 {
			if err := c.client.CommitRecords(ctx, batch...); err != nil && c.onError != nil {
				c.onError("", nil, fmt.Errorf("kafka commit: %w", err))
			}
		}
	}
}

// Close stops the consumer.
func (c *Consumer) Close() {
	c.client.Close()
}
