package pointer_constraints

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"
)

// Test event handler that captures events for verification
type testEventHandler struct {
	mu               sync.Mutex
	lockedEvents     []LockedEvent
	unlockedEvents   []UnlockedEvent
	confinedEvents   []ConfinedEvent
	unconfinedEvents []UnconfinedEvent
}

func (h *testEventHandler) HandleLocked(event LockedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lockedEvents = append(h.lockedEvents, event)
}

func (h *testEventHandler) HandleUnlocked(event UnlockedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unlockedEvents = append(h.unlockedEvents, event)
}

func (h *testEventHandler) HandleConfined(event ConfinedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.confinedEvents = append(h.confinedEvents, event)
}

func (h *testEventHandler) HandleUnconfined(event UnconfinedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unconfinedEvents = append(h.unconfinedEvents, event)
}

func (h *testEventHandler) getEventCounts() (int, int, int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.lockedEvents), len(h.unlockedEvents), len(h.confinedEvents), len(h.unconfinedEvents)
}

func (h *testEventHandler) getLastUnlockedEvent() *UnlockedEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.unlockedEvents) == 0 {
		return nil
	}
	return &h.unlockedEvents[len(h.unlockedEvents)-1]
}

func (h *testEventHandler) getLastUnconfinedEvent() *UnconfinedEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.unconfinedEvents) == 0 {
		return nil
	}
	return &h.unconfinedEvents[len(h.unconfinedEvents)-1]
}

func (h *testEventHandler) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lockedEvents = nil
	h.unlockedEvents = nil
	h.confinedEvents = nil
	h.unconfinedEvents = nil
}

// Test helper to attempt creating a manager
func tryCreateManager(t *testing.T) (*PointerConstraintsManager, func()) {
	t.Helper()
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		// Skip tests that require actual Wayland connection
		t.Skipf("Skipping test that requires Wayland connection: %v", err)
	}
	return manager, func() {
		if manager != nil {
			manager.Close()
		}
	}
}

// Basic functionality tests

func TestNewPointerConstraintsManager(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Skipf("Cannot test without Wayland: %v", err)
	}
	defer manager.Close()

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
	if manager.client == nil {
		t.Fatal("Manager client should not be nil")
	}
	if manager.manager == nil {
		t.Fatal("Manager protocol object should not be nil")
	}
}

func TestManagerClose(t *testing.T) {
	manager, cleanup := tryCreateManager(t)
	defer cleanup()

	// Test close
	err := manager.Close()
	if err != nil {
		t.Fatalf("Failed to close manager: %v", err)
	}

	// Test double close should not panic
	err = manager.Close()
	if err != nil {
		t.Logf("Second close returned error (expected): %v", err)
	}
}

func TestManagerCloseNilComponents(t *testing.T) {
	// Test close with nil components
	manager := &PointerConstraintsManager{}
	err := manager.Close()
	if err != nil {
		t.Fatalf("Close should handle nil components gracefully: %v", err)
	}
}

// Lifetime constant tests

func TestLifetimeConstants(t *testing.T) {
	// Verify that constants have different values
	if LifetimeOneshot == LifetimePersistent {
		t.Fatal("LifetimeOneshot and LifetimePersistent should have different values")
	}

	// Test that constants are reasonable values (typically 1 and 2 in Wayland protocols)
	validValues := []uint32{1, 2}

	foundOneshot := false
	foundPersistent := false

	for _, val := range validValues {
		if LifetimeOneshot == val {
			foundOneshot = true
		}
		if LifetimePersistent == val {
			foundPersistent = true
		}
	}

	if !foundOneshot {
		t.Fatalf("LifetimeOneshot should be 1 or 2, got %d", LifetimeOneshot)
	}
	if !foundPersistent {
		t.Fatalf("LifetimePersistent should be 1 or 2, got %d", LifetimePersistent)
	}
}

// Event type tests

func TestEventTypes(t *testing.T) {
	// Test LockedEvent
	lockedEvent := LockedEvent{}
	_ = lockedEvent // Ensure it compiles

	// Test UnlockedEvent
	unlockedEvent := UnlockedEvent{Lifetime: LifetimeOneshot}
	if unlockedEvent.Lifetime != LifetimeOneshot {
		t.Fatalf("Expected lifetime %d, got %d", LifetimeOneshot, unlockedEvent.Lifetime)
	}

	// Test ConfinedEvent
	confinedEvent := ConfinedEvent{}
	_ = confinedEvent // Ensure it compiles

	// Test UnconfinedEvent
	unconfinedEvent := UnconfinedEvent{Lifetime: LifetimePersistent}
	if unconfinedEvent.Lifetime != LifetimePersistent {
		t.Fatalf("Expected lifetime %d, got %d", LifetimePersistent, unconfinedEvent.Lifetime)
	}
}

// Event handler tests

func TestEventHandlerInterface(t *testing.T) {
	handler := &testEventHandler{}

	// Test that it implements EventHandler interface
	var _ EventHandler = handler

	// Test event handling
	handler.HandleLocked(LockedEvent{})
	handler.HandleUnlocked(UnlockedEvent{Lifetime: LifetimeOneshot})
	handler.HandleConfined(ConfinedEvent{})
	handler.HandleUnconfined(UnconfinedEvent{Lifetime: LifetimePersistent})

	// Verify events were captured
	locked, unlocked, confined, unconfined := handler.getEventCounts()
	if locked != 1 {
		t.Fatalf("Expected 1 locked event, got %d", locked)
	}
	if unlocked != 1 {
		t.Fatalf("Expected 1 unlocked event, got %d", unlocked)
	}
	if confined != 1 {
		t.Fatalf("Expected 1 confined event, got %d", confined)
	}
	if unconfined != 1 {
		t.Fatalf("Expected 1 unconfined event, got %d", unconfined)
	}

	// Test event data
	lastUnlocked := handler.getLastUnlockedEvent()
	if lastUnlocked == nil {
		t.Fatal("Should have unlocked event")
	}
	if lastUnlocked.Lifetime != LifetimeOneshot {
		t.Fatalf("Expected lifetime %d, got %d", LifetimeOneshot, lastUnlocked.Lifetime)
	}

	lastUnconfined := handler.getLastUnconfinedEvent()
	if lastUnconfined == nil {
		t.Fatal("Should have unconfined event")
	}
	if lastUnconfined.Lifetime != LifetimePersistent {
		t.Fatalf("Expected lifetime %d, got %d", LifetimePersistent, lastUnconfined.Lifetime)
	}

	// Test reset
	handler.reset()
	locked, unlocked, confined, unconfined = handler.getEventCounts()
	if locked != 0 || unlocked != 0 || confined != 0 || unconfined != 0 {
		t.Fatalf("Expected all counts to be 0 after reset, got locked=%d, unlocked=%d, confined=%d, unconfined=%d",
			locked, unlocked, confined, unconfined)
	}
}

<<<<<<< HEAD
// Mock types for testing
type mockSurface struct{}
type mockPointer struct{}
type mockRegion struct{}
=======
// Thread safety tests for event handler

func TestEventHandlerThreadSafety(t *testing.T) {
	eventHandler := &testEventHandler{}

	const numGoroutines = 10
	const numEvents = 100

	var wg sync.WaitGroup

	// Start multiple goroutines generating events
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numEvents; j++ {
				eventHandler.HandleLocked(LockedEvent{})
				eventHandler.HandleUnlocked(UnlockedEvent{Lifetime: LifetimeOneshot})
				eventHandler.HandleConfined(ConfinedEvent{})
				eventHandler.HandleUnconfined(UnconfinedEvent{Lifetime: LifetimePersistent})
			}
		}()
	}

	// Start another goroutine reading events
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numEvents; i++ {
			eventHandler.getEventCounts()
			eventHandler.getLastUnlockedEvent()
			eventHandler.getLastUnconfinedEvent()
			time.Sleep(time.Microsecond) // Small delay to interleave with writers
		}
	}()

	wg.Wait()

	// Verify final event counts
	locked, unlocked, confined, unconfined := eventHandler.getEventCounts()
	expectedEvents := numGoroutines * numEvents

	if locked != expectedEvents {
		t.Errorf("Expected %d locked events, got %d", expectedEvents, locked)
	}
	if unlocked != expectedEvents {
		t.Errorf("Expected %d unlocked events, got %d", expectedEvents, unlocked)
	}
	if confined != expectedEvents {
		t.Errorf("Expected %d confined events, got %d", expectedEvents, confined)
	}
	if unconfined != expectedEvents {
		t.Errorf("Expected %d unconfined events, got %d", expectedEvents, unconfined)
	}
}

// Internal event handler tests

func TestLockedPointerEventHandler(t *testing.T) {
	// Test oneshot lifetime behavior
	t.Run("oneshot lifetime", func(t *testing.T) {
		mockLP := &LockedPointer{active: true}
		handler := &lockedPointerEventHandler{
			locked:   mockLP,
			lifetime: LifetimeOneshot,
		}

		// Test locked event
		handler.HandleLocked(nil)
		if !handler.isLocked {
			t.Fatal("Handler should be in locked state")
		}

		// Test unlocked event with oneshot - should deactivate
		handler.HandleUnlocked(nil)
		if handler.isLocked {
			t.Fatal("Handler should not be in locked state after unlock")
		}
		if mockLP.active {
			t.Fatal("LockedPointer should be inactive after oneshot unlock")
		}
	})

	// Test persistent lifetime behavior
	t.Run("persistent lifetime", func(t *testing.T) {
		mockLP := &LockedPointer{active: true}
		handler := &lockedPointerEventHandler{
			locked:   mockLP,
			lifetime: LifetimePersistent,
		}

		// Test locked event
		handler.HandleLocked(nil)
		if !handler.isLocked {
			t.Fatal("Handler should be in locked state")
		}

		// Test unlocked event with persistent - should not deactivate
		handler.HandleUnlocked(nil)
		if handler.isLocked {
			t.Fatal("Handler should not be in locked state after unlock")
		}
		if !mockLP.active {
			t.Fatal("LockedPointer should remain active after persistent unlock")
		}
	})

	// Test with custom event handler
	t.Run("custom event handler", func(t *testing.T) {
		testHandler := &testEventHandler{}
		mockLP := &LockedPointer{active: true}
		internalHandler := &lockedPointerEventHandler{
			locked:       mockLP,
			lifetime:     LifetimeOneshot,
			eventHandler: testHandler,
		}

		// Test events are forwarded to custom handler
		internalHandler.HandleLocked(nil)
		internalHandler.HandleUnlocked(nil)

		locked, unlocked, _, _ := testHandler.getEventCounts()
		if locked != 1 {
			t.Fatalf("Expected 1 locked event, got %d", locked)
		}
		if unlocked != 1 {
			t.Fatalf("Expected 1 unlocked event, got %d", unlocked)
		}

		// Check unlocked event has correct lifetime
		lastUnlocked := testHandler.getLastUnlockedEvent()
		if lastUnlocked == nil || lastUnlocked.Lifetime != LifetimeOneshot {
			t.Fatal("Unlocked event should have correct lifetime")
		}
	})
}

func TestConfinedPointerEventHandler(t *testing.T) {
	// Test oneshot lifetime behavior
	t.Run("oneshot lifetime", func(t *testing.T) {
		mockCP := &ConfinedPointer{active: true}
		handler := &confinedPointerEventHandler{
			confined: mockCP,
			lifetime: LifetimeOneshot,
		}

		// Test confined event
		handler.HandleConfined(nil)
		if !handler.isConfined {
			t.Fatal("Handler should be in confined state")
		}

		// Test unconfined event with oneshot - should deactivate
		handler.HandleUnconfined(nil)
		if handler.isConfined {
			t.Fatal("Handler should not be in confined state after unconfine")
		}
		if mockCP.active {
			t.Fatal("ConfinedPointer should be inactive after oneshot unconfine")
		}
	})

	// Test persistent lifetime behavior
	t.Run("persistent lifetime", func(t *testing.T) {
		mockCP := &ConfinedPointer{active: true}
		handler := &confinedPointerEventHandler{
			confined: mockCP,
			lifetime: LifetimePersistent,
		}

		// Test confined event
		handler.HandleConfined(nil)
		if !handler.isConfined {
			t.Fatal("Handler should be in confined state")
		}

		// Test unconfined event with persistent - should not deactivate
		handler.HandleUnconfined(nil)
		if handler.isConfined {
			t.Fatal("Handler should not be in confined state after unconfine")
		}
		if !mockCP.active {
			t.Fatal("ConfinedPointer should remain active after persistent unconfine")
		}
	})

	// Test with custom event handler
	t.Run("custom event handler", func(t *testing.T) {
		testHandler := &testEventHandler{}
		mockCP := &ConfinedPointer{active: true}
		internalHandler := &confinedPointerEventHandler{
			confined:     mockCP,
			lifetime:     LifetimePersistent,
			eventHandler: testHandler,
		}

		// Test events are forwarded to custom handler
		internalHandler.HandleConfined(nil)
		internalHandler.HandleUnconfined(nil)

		_, _, confined, unconfined := testHandler.getEventCounts()
		if confined != 1 {
			t.Fatalf("Expected 1 confined event, got %d", confined)
		}
		if unconfined != 1 {
			t.Fatalf("Expected 1 unconfined event, got %d", unconfined)
		}

		// Check unconfined event has correct lifetime
		lastUnconfined := testHandler.getLastUnconfinedEvent()
		if lastUnconfined == nil || lastUnconfined.Lifetime != LifetimePersistent {
			t.Fatal("Unconfined event should have correct lifetime")
		}
	})
}

// Convenience function tests (API only)

func TestConvenienceFunctionSignatures(t *testing.T) {
	// These tests just verify the function signatures are correct
	// They will skip if Wayland is not available

	t.Run("LockPointerAtCurrentPosition", func(t *testing.T) {
		manager, cleanup := tryCreateManager(t)
		defer cleanup()

		// This will fail due to nil arguments, but tests the signature
		_, err := LockPointerAtCurrentPosition(manager, nil, nil)
		if err == nil {
			t.Fatal("Should fail with nil arguments")
		}
	})

	t.Run("LockPointerPersistent", func(t *testing.T) {
		manager, cleanup := tryCreateManager(t)
		defer cleanup()

		// This will fail due to nil arguments, but tests the signature
		_, err := LockPointerPersistent(manager, nil, nil)
		if err == nil {
			t.Fatal("Should fail with nil arguments")
		}
	})

	t.Run("ConfinePointerToRegion", func(t *testing.T) {
		manager, cleanup := tryCreateManager(t)
		defer cleanup()

		// This will fail due to nil arguments, but tests the signature
		_, err := ConfinePointerToRegion(manager, nil, nil, nil)
		if err == nil {
			t.Fatal("Should fail with nil arguments")
		}
	})

	t.Run("ConfinePointerToRegionPersistent", func(t *testing.T) {
		manager, cleanup := tryCreateManager(t)
		defer cleanup()

		// This will fail due to nil arguments, but tests the signature
		_, err := ConfinePointerToRegionPersistent(manager, nil, nil, nil)
		if err == nil {
			t.Fatal("Should fail with nil arguments")
		}
	})
}

// Mock object tests for internal structure verification

func TestLockedPointerStructure(t *testing.T) {
	// Test creating a LockedPointer structure
	lp := &LockedPointer{
		active: true,
	}

	// Test IsActive with nil handler (will panic due to nil pointer dereference)
	// This is a known limitation - the IsActive method requires a handler
	// Let's test with a proper handler instead

	// Test with handler
	handler := &lockedPointerEventHandler{isLocked: true}
	lp.handler = handler

	if !lp.IsActive() {
		t.Fatal("LockedPointer with locked handler should be active")
	}

	// Test deactivation
	lp.active = false
	if lp.IsActive() {
		t.Fatal("Inactive LockedPointer should not be active")
	}
}

func TestConfinedPointerStructure(t *testing.T) {
	// Test creating a ConfinedPointer structure
	cp := &ConfinedPointer{
		active: true,
	}

	// Test IsActive with nil handler (will panic due to nil pointer dereference)
	// This is a known limitation - the IsActive method requires a handler
	// Let's test with a proper handler instead

	// Test with handler
	handler := &confinedPointerEventHandler{isConfined: true}
	cp.handler = handler

	if !cp.IsActive() {
		t.Fatal("ConfinedPointer with confined handler should be active")
	}

	// Test deactivation
	cp.active = false
	if cp.IsActive() {
		t.Fatal("Inactive ConfinedPointer should not be active")
	}
}

// Thread safety tests for internal handlers

func TestInternalHandlerThreadSafety(t *testing.T) {
	t.Run("locked pointer handler", func(t *testing.T) {
		mockLP := &LockedPointer{active: true}
		handler := &lockedPointerEventHandler{
			locked:   mockLP,
			lifetime: LifetimeOneshot,
		}

		const numGoroutines = 10
		const numOperations = 100

		var wg sync.WaitGroup

		// Test concurrent access to handler
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					handler.HandleLocked(nil)
					handler.HandleUnlocked(nil)
				}
			}()
		}

		wg.Wait()
		// If we get here without deadlock or race conditions, the test passes
	})

	t.Run("confined pointer handler", func(t *testing.T) {
		mockCP := &ConfinedPointer{active: true}
		handler := &confinedPointerEventHandler{
			confined: mockCP,
			lifetime: LifetimeOneshot,
		}

		const numGoroutines = 10
		const numOperations = 100

		var wg sync.WaitGroup

		// Test concurrent access to handler
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					handler.HandleConfined(nil)
					handler.HandleUnconfined(nil)
				}
			}()
		}

		wg.Wait()
		// If we get here without deadlock or race conditions, the test passes
	})
}

// Memory allocation tests

func TestMemoryAllocation(t *testing.T) {
	// Test that we don't have obvious memory leaks in event handling
	handler := &testEventHandler{}

	runtime.GC()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	const iterations = 1000
	for i := 0; i < iterations; i++ {
		handler.HandleLocked(LockedEvent{})
		handler.HandleUnlocked(UnlockedEvent{Lifetime: LifetimeOneshot})
		handler.HandleConfined(ConfinedEvent{})
		handler.HandleUnconfined(UnconfinedEvent{Lifetime: LifetimePersistent})

		// Reset periodically to prevent unbounded growth
		if i%100 == 0 {
			handler.reset()
		}
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// This is a rough check - we allow some growth but not excessive
	if m2.Alloc > m1.Alloc*3 && m2.Alloc-m1.Alloc > 1024*1024 {
		t.Logf("Memory usage grew from %d to %d bytes", m1.Alloc, m2.Alloc)
		t.Logf("This might indicate a memory leak, but could also be normal")
	}
}

// Benchmark tests

func BenchmarkEventHandling(b *testing.B) {
	handler := &testEventHandler{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleLocked(LockedEvent{})
		handler.HandleUnlocked(UnlockedEvent{Lifetime: LifetimeOneshot})
	}
}

func BenchmarkEventHandlerConcurrent(b *testing.B) {
	handler := &testEventHandler{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			handler.HandleLocked(LockedEvent{})
			handler.HandleUnlocked(UnlockedEvent{Lifetime: LifetimeOneshot})
		}
	})
}

func BenchmarkInternalHandlerEvents(b *testing.B) {
	mockLP := &LockedPointer{active: true}
	handler := &lockedPointerEventHandler{
		locked:   mockLP,
		lifetime: LifetimeOneshot,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.HandleLocked(nil)
		handler.HandleUnlocked(nil)
	}
}
>>>>>>> c07acb9 (test: add comprehensive pointer constraints tests)
