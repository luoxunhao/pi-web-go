package session

import (
	"testing"
	"time"

	"github.com/luoxunhao/pi-web-go/internal/pigo"
)

func TestObserveEventRunningAndIdle(t *testing.T) {
	m := NewManager(time.Minute)
	m.ObserveEvent(pigo.DomainEvent{
		Type: "session.status",
		Data: map[string]interface{}{"sessionId": "s1", "directory": "/work", "status": "running"},
	})
	ids := m.RunningIDs()
	if len(ids) != 1 || ids[0] != "s1" {
		t.Fatalf("running = %v", ids)
	}
	m.ObserveEvent(pigo.DomainEvent{
		Type: "session.status",
		Data: map[string]interface{}{"sessionId": "s1", "status": "idle"},
	})
	if got := m.RunningIDs(); len(got) != 0 {
		t.Fatalf("running after idle = %v", got)
	}
}

func TestQueueAndDeltaKeepRunning(t *testing.T) {
	m := NewManager(time.Minute)
	m.ObserveEvent(pigo.DomainEvent{
		Type: "queue.updated",
		Data: map[string]interface{}{"sessionId": "s1", "directory": "/work", "queuedCount": 2},
	})
	if got := m.RunningIDs(); len(got) != 1 {
		t.Fatalf("running = %v", got)
	}
	m.ObserveEvent(pigo.DomainEvent{
		Type: "message.part.delta",
		Data: map[string]interface{}{"sessionId": "s1", "messageId": "m1", "partId": "text", "delta": "x"},
	})
	if got := m.RunningIDs(); len(got) != 1 {
		t.Fatalf("running after delta = %v", got)
	}
}

func TestCleanupMarksInactive(t *testing.T) {
	m := NewManager(10 * time.Minute)
	m.ObserveEvent(pigo.DomainEvent{
		Type: "session.status",
		Data: map[string]interface{}{"sessionId": "s1", "status": "running"},
	})
	m.Cleanup(time.Now().Add(11 * time.Minute))
	if got := m.RunningIDs(); len(got) != 0 {
		t.Fatalf("running after cleanup = %v", got)
	}
}

func TestSubscribeReceivesUpdates(t *testing.T) {
	m := NewManager(time.Minute)
	ch, unsubscribe := m.Subscribe()
	defer unsubscribe()
	m.MarkRunning("s1", "/work", "m1")
	select {
	case ids := <-ch:
		if len(ids) != 1 || ids[0] != "s1" {
			t.Fatalf("ids = %v", ids)
		}
	case <-time.After(time.Second):
		t.Fatal("no update received")
	}
}
