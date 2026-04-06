package engine

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType identifies the kind of engine event.
type EventType string

const (
	EventStepStarted        EventType = "step_started"
	EventStepChunk          EventType = "step_chunk"
	EventStepCompleted      EventType = "step_completed"
	EventStepFailed         EventType = "step_failed"
	EventExecutionCompleted EventType = "execution_completed"
	EventExecutionFailed    EventType = "execution_failed"
)

// Event is a single engine event published during execution.
type Event struct {
	Type        EventType       `json:"type"`
	ExecutionID uuid.UUID       `json:"execution_id"`
	NodeID      string          `json:"node_id,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
	Error       string          `json:"error,omitempty"`
	Timestamp   time.Time       `json:"timestamp"`
}

// EventBus provides pub/sub for execution events.
// Subscribers receive events on a buffered channel keyed by execution ID.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]chan Event
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[uuid.UUID][]chan Event),
	}
}

// Subscribe returns a channel that will receive events for the given execution.
// The channel is buffered to avoid blocking the publisher.
func (b *EventBus) Subscribe(executionID uuid.UUID) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 64)
	b.subscribers[executionID] = append(b.subscribers[executionID], ch)
	return ch
}

// Unsubscribe removes a channel from the subscriber list and closes it.
func (b *EventBus) Unsubscribe(executionID uuid.UUID, ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[executionID]
	filtered := make([]chan Event, 0, len(subs))
	for _, s := range subs {
		if (<-chan Event)(s) == ch {
			close(s)
		} else {
			filtered = append(filtered, s)
		}
	}
	if len(filtered) == 0 {
		delete(b.subscribers, executionID)
	} else {
		b.subscribers[executionID] = filtered
	}
}

// Publish sends an event to all subscribers of the given execution.
// Non-blocking: if a subscriber's buffer is full, the event is dropped for that subscriber.
func (b *EventBus) Publish(executionID uuid.UUID, event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers[executionID] {
		select {
		case ch <- event:
		default:
			// subscriber buffer full, drop event
		}
	}
}
