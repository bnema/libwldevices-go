package pointer_constraints

import (
	"context"
	"testing"
)

// Test lifetime constants
func TestLifetimeConstants(t *testing.T) {
	// Verify that constants have different values
	if LifetimeOneshot == LifetimePersistent {
		t.Fatal("LifetimeOneshot and LifetimePersistent should have different values")
	}

	// Verify correct values
	if LifetimeOneshot != 1 {
		t.Errorf("LifetimeOneshot should be 1, got %d", LifetimeOneshot)
	}

	if LifetimePersistent != 2 {
		t.Errorf("LifetimePersistent should be 2, got %d", LifetimePersistent)
	}

	// Test alternative names
	if LIFETIME_ONESHOT != LifetimeOneshot {
		t.Fatal("LIFETIME_ONESHOT should equal LifetimeOneshot")
	}

	if LIFETIME_PERSISTENT != LifetimePersistent {
		t.Fatal("LIFETIME_PERSISTENT should equal LifetimePersistent")
	}
}

// Test error constants
func TestErrorConstants(t *testing.T) {
	if ERROR_ALREADY_CONSTRAINED != 1 {
		t.Errorf("ERROR_ALREADY_CONSTRAINED should be 1, got %d", ERROR_ALREADY_CONSTRAINED)
	}
}

// Test PointerConstraintsError
func TestPointerConstraintsError(t *testing.T) {
	err := &PointerConstraintsError{
		Code:    ERROR_ALREADY_CONSTRAINED,
		Message: "test error",
	}

	expected := "pointer constraints error 1: test error"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

// Test NewPointerConstraintsManager with no Wayland
func TestNewPointerConstraintsManager(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Skipf("Cannot test without Wayland: %v", err)
	}
	defer func() {
		if manager != nil {
			_ = manager.Close()
		}
	}()

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
}

// Test manager operations with invalid arguments
func TestManagerInvalidArguments(t *testing.T) {
	// Create a mock manager for testing (this would fail to connect but we can test the struct)
	manager := &PointerConstraintsManager{}

	// Test LockPointer with nil manager
	_, err := manager.LockPointer(nil, nil, nil, LifetimeOneshot)
	if err == nil {
		t.Fatal("LockPointer should fail with nil internal manager")
	}

	// Test ConfinePointer with nil manager
	_, err = manager.ConfinePointer(nil, nil, nil, LifetimeOneshot)
	if err == nil {
		t.Fatal("ConfinePointer should fail with nil internal manager")
	}

	// Test invalid lifetime
	manager.manager = nil // Ensure it's nil for this test
	_, err = manager.LockPointer(nil, nil, nil, 999)
	if err == nil {
		t.Fatal("LockPointer should fail with invalid lifetime")
	}

	_, err = manager.ConfinePointer(nil, nil, nil, 999)
	if err == nil {
		t.Fatal("ConfinePointer should fail with invalid lifetime")
	}
}

// Test LockedPointer operations with nil pointer
func TestLockedPointerNilOperations(t *testing.T) {
	lp := &LockedPointer{}

	// Test Destroy with nil locked pointer
	err := lp.Destroy()
	if err != nil {
		t.Errorf("Destroy should handle nil locked pointer gracefully, got error: %v", err)
	}

	// Test SetCursorPositionHint with nil locked pointer
	err = lp.SetCursorPositionHint(100.0, 200.0)
	if err == nil {
		t.Fatal("SetCursorPositionHint should fail with nil locked pointer")
	}

	// Test SetRegion with nil locked pointer
	err = lp.SetRegion(nil)
	if err == nil {
		t.Fatal("SetRegion should fail with nil locked pointer")
	}
}

// Test ConfinedPointer operations with nil pointer
func TestConfinedPointerNilOperations(t *testing.T) {
	cp := &ConfinedPointer{}

	// Test Destroy with nil confined pointer
	err := cp.Destroy()
	if err != nil {
		t.Errorf("Destroy should handle nil confined pointer gracefully, got error: %v", err)
	}

	// Test SetRegion with nil confined pointer
	err = cp.SetRegion(nil)
	if err == nil {
		t.Fatal("SetRegion should fail with nil confined pointer")
	}
}

// Test convenience functions
func TestConvenienceFunctions(t *testing.T) {
	manager := &PointerConstraintsManager{} // nil internal manager for testing

	// Test LockPointerAtCurrentPosition
	_, err := LockPointerAtCurrentPosition(manager, nil, nil)
	if err == nil {
		t.Fatal("LockPointerAtCurrentPosition should fail with nil internal manager")
	}

	// Test LockPointerPersistent
	_, err = LockPointerPersistent(manager, nil, nil)
	if err == nil {
		t.Fatal("LockPointerPersistent should fail with nil internal manager")
	}

	// Test ConfinePointerToRegion
	_, err = ConfinePointerToRegion(manager, nil, nil, nil)
	if err == nil {
		t.Fatal("ConfinePointerToRegion should fail with nil internal manager")
	}
}

// Test manager Close operations
func TestManagerClose(t *testing.T) {
	manager := &PointerConstraintsManager{}

	// Test close with nil components
	err := manager.Close()
	if err != nil {
		t.Errorf("Close should handle nil components gracefully, got error: %v", err)
	}

	// Test Destroy (which calls Close)
	err = manager.Destroy()
	if err != nil {
		t.Errorf("Destroy should handle nil components gracefully, got error: %v", err)
	}
}

// Benchmark basic operations
func BenchmarkLifetimeConstants(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = LifetimeOneshot
		_ = LifetimePersistent
	}
}

func BenchmarkErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := &PointerConstraintsError{
			Code:    ERROR_ALREADY_CONSTRAINED,
			Message: "benchmark error",
		}
		_ = err.Error()
	}
}