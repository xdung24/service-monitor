package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/xdung24/conductor/internal/models"
)

// KafkaChecker produces a single test message to the configured Kafka topic.
// A successful produce (no error) indicates the broker is reachable and the
// topic is writable.
//
// URL format:  broker:9092  (plain TCP; TLS not yet supported)
// Topic:       stored in m.KafkaTopic (required)
type KafkaChecker struct{}

// Check produces a test message to the configured Kafka topic.
func (c *KafkaChecker) Check(ctx context.Context, m *models.Monitor) Result {
	broker := strings.TrimSpace(m.URL)
	if broker == "" {
		return Result{Status: 0, Message: "broker address is required (URL field)"}
	}

	topic := strings.TrimSpace(m.KafkaTopic)
	if topic == "" {
		topic = "conductor-healthcheck"
	}

	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	if timeout < time.Second {
		timeout = 10 * time.Second
	}

	writeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{broker},
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  1,
		BatchTimeout: timeout,
		Async:        false,
	})
	defer w.Close() //nolint:errcheck

	err := w.WriteMessages(writeCtx, kafka.Message{
		Key:   []byte("conductor"),
		Value: []byte(fmt.Sprintf(`{"check":"healthcheck","time":%d}`, time.Now().Unix())),
	})
	if err != nil {
		return Result{Status: 0, Message: "produce failed: " + err.Error()}
	}

	return Result{Status: 1, Message: fmt.Sprintf("produced to topic %q successfully", topic)}
}
