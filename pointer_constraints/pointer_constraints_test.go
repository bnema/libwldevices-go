package pointer_constraints

import (
	"context"
	"testing"
)

func TestNewPointerConstraintsManager(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create pointer constraints manager: %v", err)
	}
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
}

func TestManagerDestroy(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	err = manager.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy manager: %v", err)
	}

	// Second destroy should fail
	err = manager.Destroy()
	if err == nil {
		t.Fatal("Second destroy should fail")
	}
}

func TestLockPointer(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Destroy()

	surface := &mockSurface{}
	pointer := &mockPointer{}

	// Test oneshot lifetime
	lockedPointer, err := manager.LockPointer(surface, pointer, nil, LIFETIME_ONESHOT)
	if err != nil {
		t.Fatalf("Failed to lock pointer: %v", err)
	}
	if lockedPointer == nil {
		t.Fatal("Locked pointer should not be nil")
	}

	// Test persistent lifetime
	lockedPointer2, err := manager.LockPointer(surface, pointer, nil, LIFETIME_PERSISTENT)
	if err != nil {
		t.Fatalf("Failed to lock pointer with persistent lifetime: %v", err)
	}
	if lockedPointer2 == nil {
		t.Fatal("Locked pointer should not be nil")
	}

	// Test invalid lifetime
	_, err = manager.LockPointer(surface, pointer, nil, 999)
	if err == nil {
		t.Fatal("Should fail with invalid lifetime")
	}
}

func TestConfinePointer(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Destroy()

	surface := &mockSurface{}
	pointer := &mockPointer{}
	region := &mockRegion{}

	confinedPointer, err := manager.ConfinePointer(surface, pointer, region, LIFETIME_ONESHOT)
	if err != nil {
		t.Fatalf("Failed to confine pointer: %v", err)
	}
	if confinedPointer == nil {
		t.Fatal("Confined pointer should not be nil")
	}
}

func TestLockedPointerOperations(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Destroy()

	surface := &mockSurface{}
	pointer := &mockPointer{}

	lockedPointer, err := manager.LockPointer(surface, pointer, nil, LIFETIME_ONESHOT)
	if err != nil {
		t.Fatalf("Failed to lock pointer: %v", err)
	}

	// Test cursor position hint
	err = lockedPointer.SetCursorPositionHint(100.0, 200.0)
	if err != nil {
		t.Fatalf("Failed to set cursor position hint: %v", err)
	}

	// Test set region
	region := &mockRegion{}
	err = lockedPointer.SetRegion(region)
	if err != nil {
		t.Fatalf("Failed to set region: %v", err)
	}

	// Test destroy
	err = lockedPointer.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy locked pointer: %v", err)
	}

	// Operations after destroy should fail
	err = lockedPointer.SetCursorPositionHint(0, 0)
	if err == nil {
		t.Fatal("Should fail after destroy")
	}
}

func TestConfinedPointerOperations(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Destroy()

	surface := &mockSurface{}
	pointer := &mockPointer{}
	region := &mockRegion{}

	confinedPointer, err := manager.ConfinePointer(surface, pointer, region, LIFETIME_ONESHOT)
	if err != nil {
		t.Fatalf("Failed to confine pointer: %v", err)
	}

	// Test set region
	newRegion := &mockRegion{}
	err = confinedPointer.SetRegion(newRegion)
	if err != nil {
		t.Fatalf("Failed to set region: %v", err)
	}

	// Test destroy
	err = confinedPointer.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy confined pointer: %v", err)
	}

	// Operations after destroy should fail
	err = confinedPointer.SetRegion(region)
	if err == nil {
		t.Fatal("Should fail after destroy")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	ctx := context.Background()
	manager, err := NewPointerConstraintsManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer manager.Destroy()

	surface := &mockSurface{}
	pointer := &mockPointer{}
	region := &mockRegion{}

	// Test LockPointerAtCurrentPosition
	lockedPointer, err := LockPointerAtCurrentPosition(manager, surface, pointer)
	if err != nil {
		t.Fatalf("Failed to lock pointer at current position: %v", err)
	}
	lockedPointer.Destroy()

	// Test LockPointerPersistent
	lockedPointer2, err := LockPointerPersistent(manager, surface, pointer)
	if err != nil {
		t.Fatalf("Failed to lock pointer persistent: %v", err)
	}
	lockedPointer2.Destroy()

	// Test ConfinePointerToRegion
	confinedPointer, err := ConfinePointerToRegion(manager, surface, pointer, region)
	if err != nil {
		t.Fatalf("Failed to confine pointer to region: %v", err)
	}
	confinedPointer.Destroy()
}

// Mock types for testing
type mockSurface struct{}
type mockPointer struct{}
type mockRegion struct{}