package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type JobQueue struct {
	client *redis.Client
	prefix string
}

func NewJobQueue(client *redis.Client, env string) *JobQueue {
	return &JobQueue{
		client: client,
		prefix: fmt.Sprintf("%s:schooldesk:queue", env),
	}
}

func (q *JobQueue) streamKey(queueType string) string {
	return fmt.Sprintf("%s:%s", q.prefix, queueType)
}

func (q *JobQueue) Enqueue(ctx context.Context, queueType string, payload map[string]interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey(queueType),
		Values: map[string]interface{}{
			"payload": string(raw),
			"ts":      time.Now().UTC().Format(time.RFC3339),
		},
	}).Err()
}

func (q *JobQueue) Consume(queueType string, handler func(map[string]interface{}) error) error {
	stream := q.streamKey(queueType)
	lastID := "0"
	ctx := context.Background()
	for {
		result, err := q.client.XRead(ctx, &redis.XReadArgs{
			Streams: []string{stream, lastID},
			Block:   5 * time.Second,
			Count:   10,
		}).Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return err
		}
		for _, res := range result {
			for _, msg := range res.Messages {
				raw, _ := msg.Values["payload"].(string)
				payload := map[string]interface{}{}
				if err := json.Unmarshal([]byte(raw), &payload); err != nil {
					log.Printf("invalid queue payload %s: %v", msg.ID, err)
				} else if err := handler(payload); err != nil {
					log.Printf("queue handler error %s: %v", msg.ID, err)
				}
				lastID = msg.ID
			}
		}
	}
}
