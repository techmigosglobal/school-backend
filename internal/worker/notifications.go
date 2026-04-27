package worker

import (
	"log"

	"school-backend/internal/services"
)

func RunNotificationWorker() error {
	if services.Queue == nil {
		return nil
	}
	log.Println("Notification worker started")
	return services.Queue.Consume("notifications", func(payload map[string]interface{}) error {
		log.Printf("processed notification job: %v", payload)
		return nil
	})
}
