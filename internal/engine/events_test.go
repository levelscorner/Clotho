package engine

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEventBus_Subscribe(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	execID := uuid.New()
	ch := bus.Subscribe(execID)

	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}
}

func TestEventBus_Publish(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	execID := uuid.New()
	ch := bus.Subscribe(execID)

	event := Event{
		Type:        EventStepStarted,
		ExecutionID: execID,
		NodeID:      "node-1",
		Timestamp:   time.Now(),
	}

	bus.Publish(execID, event)

	select {
	case got := <-ch:
		if got.Type != EventStepStarted {
			t.Errorf("event Type = %q, want %q", got.Type, EventStepStarted)
		}
		if got.NodeID != "node-1" {
			t.Errorf("event NodeID = %q, want %q", got.NodeID, "node-1")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestEventBus_PublishToNonExistentExecution(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	execID := uuid.New()
	event := Event{
		Type:        EventStepStarted,
		ExecutionID: execID,
		Timestamp:   time.Now(),
	}

	// Should not panic
	bus.Publish(execID, event)
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	execID := uuid.New()
	ch1 := bus.Subscribe(execID)
	ch2 := bus.Subscribe(execID)

	event := Event{
		Type:        EventStepCompleted,
		ExecutionID: execID,
		NodeID:      "node-1",
		Timestamp:   time.Now(),
	}

	bus.Publish(execID, event)

	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case got := <-ch:
			if got.Type != EventStepCompleted {
				t.Errorf("subscriber %d: event Type = %q, want %q", i, got.Type, EventStepCompleted)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out waiting for event", i)
		}
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	execID := uuid.New()
	ch := bus.Subscribe(execID)

	bus.Unsubscribe(execID, ch)

	// Channel should be closed after unsubscribe
	_, open := <-ch
	if open {
		t.Error("expected channel to be closed after Unsubscribe")
	}

	// Publishing after unsubscribe should not panic
	event := Event{
		Type:        EventStepStarted,
		ExecutionID: execID,
		Timestamp:   time.Now(),
	}
	bus.Publish(execID, event)
}

func TestEventBus_UnsubscribeStopsDelivery(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	execID := uuid.New()
	ch1 := bus.Subscribe(execID)
	ch2 := bus.Subscribe(execID)

	bus.Unsubscribe(execID, ch1)

	event := Event{
		Type:        EventStepStarted,
		ExecutionID: execID,
		Timestamp:   time.Now(),
	}
	bus.Publish(execID, event)

	// ch1 is closed, ch2 should still receive
	select {
	case got := <-ch2:
		if got.Type != EventStepStarted {
			t.Errorf("ch2 event Type = %q, want %q", got.Type, EventStepStarted)
		}
	case <-time.After(time.Second):
		t.Fatal("ch2: timed out waiting for event")
	}
}

func TestEventBus_BufferOverflow(t *testing.T) {
	t.Parallel()

	bus := NewEventBus()
	execID := uuid.New()
	_ = bus.Subscribe(execID) // subscribe but don't read

	event := Event{
		Type:        EventStepChunk,
		ExecutionID: execID,
		Timestamp:   time.Now(),
	}

	// Publish more than the buffer size (64) without reading.
	// Should not panic or block.
	for i := 0; i < 100; i++ {
		bus.Publish(execID, event)
	}
}
