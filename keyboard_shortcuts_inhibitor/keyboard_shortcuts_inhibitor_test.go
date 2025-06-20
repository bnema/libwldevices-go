package keyboard_shortcuts_inhibitor

import (
	"context"
	"testing"
)

func TestNewKeyboardShortcutsInhibitorManager(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create keyboard shortcuts inhibitor manager: %v", err)
	}
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
}

func TestManagerDestroy(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
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

func TestInhibitShortcuts(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Destroy() }()

	surface := &mockSurface{}
	seat := &mockSeat{}

	inhibitor, err := manager.InhibitShortcuts(surface, seat)
	if err != nil {
		t.Fatalf("Failed to inhibit shortcuts: %v", err)
	}
	if inhibitor == nil {
		t.Fatal("Inhibitor should not be nil")
	}

	// Test that the inhibitor is active
	status := GetStatus(inhibitor)
	if !status.Active {
		t.Fatal("Inhibitor should be active")
	}
	if status.Surface != surface {
		t.Fatal("Surface should match")
	}
	if status.Seat != seat {
		t.Fatal("Seat should match")
	}
}

func TestInhibitShortcutsWithNilSurface(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Destroy() }()

	seat := &mockSeat{}

	_, err = manager.InhibitShortcuts(nil, seat)
	if err == nil {
		t.Fatal("Should fail with nil surface")
	}
}

func TestInhibitShortcutsWithNilSeat(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Destroy() }()

	surface := &mockSurface{}

	_, err = manager.InhibitShortcuts(surface, nil)
	if err == nil {
		t.Fatal("Should fail with nil seat")
	}
}

func TestInhibitorDestroy(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Destroy() }()

	surface := &mockSurface{}
	seat := &mockSeat{}

	inhibitor, err := manager.InhibitShortcuts(surface, seat)
	if err != nil {
		t.Fatalf("Failed to inhibit shortcuts: %v", err)
	}

	// Test destroy
	err = inhibitor.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy inhibitor: %v", err)
	}

	// Test that inhibitor is no longer active
	status := GetStatus(inhibitor)
	if status.Active {
		t.Fatal("Inhibitor should not be active after destroy")
	}

	// Second destroy should fail
	err = inhibitor.Destroy()
	if err == nil {
		t.Fatal("Second destroy should fail")
	}
}

func TestCreateTemporaryInhibitor(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}
	defer func() { _ = manager.Destroy() }()

	surface := &mockSurface{}
	seat := &mockSeat{}

	inhibitor, err := CreateTemporaryInhibitor(manager, surface, seat)
	if err != nil {
		t.Fatalf("Failed to create temporary inhibitor: %v", err)
	}
	if inhibitor == nil {
		t.Fatal("Inhibitor should not be nil")
	}

	// Verify it works
	status := GetStatus(inhibitor)
	if !status.Active {
		t.Fatal("Temporary inhibitor should be active")
	}

	// Clean up
	_ = inhibitor.Destroy()
}

func TestGetStatusWithInvalidInhibitor(t *testing.T) {
	// Test GetStatus with a non-implementation type
	fakeInhibitor := &fakeInhibitor{}
	status := GetStatus(fakeInhibitor)
	if status.Active {
		t.Fatal("Status should show inactive for non-implementation types")
	}
}

func TestInhibitorAfterManagerDestroy(t *testing.T) {
	ctx := context.Background()
	manager, err := NewKeyboardShortcutsInhibitorManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	surface := &mockSurface{}
	seat := &mockSeat{}

	inhibitor, err := manager.InhibitShortcuts(surface, seat)
	if err != nil {
		t.Fatalf("Failed to inhibit shortcuts: %v", err)
	}

	// Destroy manager
	_ = manager.Destroy()

	// Try to create new inhibitor after manager is destroyed
	_, err = manager.InhibitShortcuts(surface, seat)
	if err == nil {
		t.Fatal("Should fail to create inhibitor after manager is destroyed")
	}

	// Existing inhibitor should still work
	err = inhibitor.Destroy()
	if err != nil {
		t.Fatalf("Existing inhibitor should still be destroyable: %v", err)
	}
}

// Mock types for testing
type mockSurface struct{}
type mockSeat struct{}

// Fake inhibitor for testing GetStatus with non-implementation types
type fakeInhibitor struct{}

func (f *fakeInhibitor) Destroy() error {
	return nil
}