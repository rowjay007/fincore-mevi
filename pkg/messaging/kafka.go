package messaging

import (
	"context"
	"time"

	"fincore/pkg/outbox/relay"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
}

func NewKafkaPublisher(brokers []string) *KafkaPublisher {
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.LeastBytes{},
			MaxAttempts:  5,
			BatchSize:    100,
			BatchTimeout: 10 * time.Millisecond,
			Async:        false, // For reliability in outbox relay
		},
	}
}

func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key []byte, payload []byte, headers map[string]string) error {
	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   payload,
		Headers: kafkaHeaders,
	})
}

func (p *KafkaPublisher) Close() error {
	return p.writer.Close()
}

// Ensure KafkaPublisher implements relay.Publisher
var _ relay.Publisher = (*KafkaPublisher)(nil)
