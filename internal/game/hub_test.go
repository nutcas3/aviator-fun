package game

import (
	"sync"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()

	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}

	if hub.clients == nil {
		t.Error("Hub clients map is nil")
	}

	if hub.broadcast == nil {
		t.Error("Hub broadcast channel is nil")
	}

	if hub.register == nil {
		t.Error("Hub register channel is nil")
	}

	if hub.unregister == nil {
		t.Error("Hub unregister channel is nil")
	}
}

func TestHub_GetClientCount(t *testing.T) {
	hub := NewHub()

	// Initial count should be 0
	if count := hub.GetClientCount(); count != 0 {
		t.Errorf("GetClientCount() = %v, want 0", count)
	}
}

func TestHub_Broadcast(t *testing.T) {
	hub := NewHub()

	// Start the hub
	go hub.Run()
	defer func() {
		// Clean shutdown
		close(hub.broadcast)
	}()

	// Give the hub time to start
	time.Sleep(10 * time.Millisecond)

	// Test broadcasting a message
	message := map[string]interface{}{
		"type": "test",
		"data": "test_data",
	}

	// Should not block
	hub.Broadcast(message)

	// Give time for broadcast to process
	time.Sleep(10 * time.Millisecond)
}

func TestHub_BroadcastChannelFull(t *testing.T) {
	hub := NewHub()

	// Don't start the hub, so broadcast channel fills up
	// Fill the channel (capacity is 100)
	for i := 0; i < 100; i++ {
		hub.Broadcast(map[string]string{"msg": "test"})
	}

	// Next broadcast should not block (should drop message)
	done := make(chan bool, 1)
	go func() {
		hub.Broadcast(map[string]string{"msg": "overflow"})
		done <- true
	}()

	select {
	case <-done:
		// Success - didn't block
	case <-time.After(100 * time.Millisecond):
		t.Error("Broadcast() blocked when channel was full")
	}
}

func TestHub_ConcurrentBroadcasts(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.broadcast)

	time.Sleep(10 * time.Millisecond)

	// Concurrent broadcasts
	var wg sync.WaitGroup
	broadcasts := 100

	for i := 0; i < broadcasts; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			hub.Broadcast(map[string]interface{}{
				"type":  "test",
				"value": n,
			})
		}(i)
	}

	// Wait for all broadcasts to complete
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Error("Concurrent broadcasts timed out")
	}
}

func TestHub_GetClientCount_ThreadSafe(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.broadcast)

	time.Sleep(10 * time.Millisecond)

	// Concurrent reads
	var wg sync.WaitGroup
	reads := 100

	for i := 0; i < reads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = hub.GetClientCount()
		}()
	}

	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// Success - no race conditions
	case <-time.After(1 * time.Second):
		t.Error("Concurrent GetClientCount() timed out")
	}
}

func BenchmarkHub_Broadcast(b *testing.B) {
	hub := NewHub()
	go hub.Run()
	defer close(hub.broadcast)

	time.Sleep(10 * time.Millisecond)

	message := map[string]interface{}{
		"type": "benchmark",
		"data": "test_data",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.Broadcast(message)
	}
}

func BenchmarkHub_GetClientCount(b *testing.B) {
	hub := NewHub()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.GetClientCount()
	}
}
