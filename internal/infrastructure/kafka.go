package infrastructure

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/IBM/sarama"
)

// KafkaProducer handles sending messages to Kafka
type KafkaProducer struct {
	producer sarama.SyncProducer
	enabled  bool
}

// MediaUploadEvent represents a media upload event
type MediaUploadEvent struct {
	UserID      int64     `json:"user_id"`
	ListID      int64     `json:"list_id"`
	ItemID      int64     `json:"item_id"`
	MediaType   string    `json:"media_type"` // "image" or "video"
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"` // Temporary local path
	S3Bucket    string    `json:"s3_bucket"`
	S3Key       string    `json:"s3_key"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// NewKafkaProducer creates a new Kafka producer
func NewKafkaProducer() *KafkaProducer {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Retry.Max = 3
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Compression = sarama.CompressionSnappy

	producer, err := sarama.NewSyncProducer([]string{brokers}, config)
	if err != nil {
		log.Printf("⚠️ Failed to create Kafka producer: %v. Media uploads will be disabled.", err)
		return &KafkaProducer{enabled: false}
	}

	log.Println("✅ Kafka producer created successfully")
	return &KafkaProducer{
		producer: producer,
		enabled:  true,
	}
}

// SendMediaUploadEvent sends a media upload event to Kafka
func (k *KafkaProducer) SendMediaUploadEvent(ctx context.Context, event MediaUploadEvent) error {
	if !k.enabled {
		log.Println("⚠️ Kafka is disabled, skipping media upload event")
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	topic := os.Getenv("KAFKA_MEDIA_TOPIC")
	if topic == "" {
		topic = "media-uploads"
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
		Key:   sarama.StringEncoder(event.S3Key),
	}

	partition, offset, err := k.producer.SendMessage(msg)
	if err != nil {
		log.Printf("❌ Failed to send Kafka message: %v", err)
		return err
	}

	log.Printf("✅ Kafka message sent to partition %d at offset %d", partition, offset)
	return nil
}

// Publish sends a generic message to a Kafka topic
func (k *KafkaProducer) Publish(topic string, data []byte) error {
	if !k.enabled {
		log.Printf("⚠️ Kafka is disabled, skipping publish to topic: %s", topic)
		return nil
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
	}

	_, _, err := k.producer.SendMessage(msg)
	if err != nil {
		log.Printf("❌ Failed to publish to topic %s: %v", topic, err)
	}
	return err
}

// Close closes the Kafka producer
func (k *KafkaProducer) Close() error {
	if !k.enabled || k.producer == nil {
		return nil
	}
	return k.producer.Close()
}
