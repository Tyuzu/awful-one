// dropify/filedrop/mq_adapter.go

package filedrop

import (
	"context"
	"encoding/json"
	"log"
	"naevis/infra"
)

// MQAdapter provides a simple MQ notification interface using the infra NATS
type MQAdapter struct {
	app *infra.Deps
}

// NewMQAdapter creates a new MQ adapter
func NewMQAdapter(app *infra.Deps) *MQAdapter {
	return &MQAdapter{app: app}
}

// Notify publishes a notification event via NATS
func (mq *MQAdapter) Notify(subject string, data interface{}) {
	if mq.app == nil || mq.app.MQ == nil {
		log.Printf("[MQAdapter] Warning: Cannot notify %s - MQ not available", subject)
		return
	}

	// Publish to NATS - convert data to JSON bytes
	ctx := context.Background()
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("[MQAdapter] Error marshaling data for %s: %v", subject, err)
		return
	}

	if err := mq.app.MQ.Publish(ctx, subject, jsonData); err != nil {
		log.Printf("[MQAdapter] Error publishing to %s: %v", subject, err)
	}
}

// Global MQ instance (will be set during initialization)
var globalMQ *MQAdapter

// SetMQAdapter sets the global MQ adapter instance
func SetMQAdapter(mq *MQAdapter) {
	globalMQ = mq
}

// GetMQAdapter returns the global MQ adapter (or creates a stub if not set)
func GetMQAdapter() *MQAdapter {
	if globalMQ == nil {
		log.Println("[MQAdapter] Warning: MQ adapter not initialized, using stub")
		return &MQAdapter{}
	}
	return globalMQ
}

// NotifyEvent is a convenience function that uses the global MQ adapter
func NotifyEvent(subject string, data interface{}) {
	GetMQAdapter().Notify(subject, data)
}
