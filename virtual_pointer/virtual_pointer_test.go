package virtual_pointer

import (
	"context"
	"testing"
	"time"
)

func TestNewVirtualPointerManager(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	// Test manager closure
	err = manager.Close()
	if err != nil {
		t.Fatalf("Failed to close manager: %v", err)
	}
}

func TestVirtualPointerCreation(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	// Test creating virtual pointer
	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	if pointer == nil {
		t.Fatal("Pointer should not be nil")
	}

	// Clean up
	_ = pointer.Close()
}

func TestVirtualPointerMotion(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer func() { _ = pointer.Close() }()

	// Test relative motion
	err = pointer.Motion(time.Now(), 10.0, 20.0)
	if err != nil {
		t.Fatalf("Failed to send motion: %v", err)
	}

	// Test absolute motion
	err = pointer.MotionAbsolute(time.Now(), 100, 200, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to send absolute motion: %v", err)
	}

	// Test convenience method for relative motion
	err = pointer.MoveRelative(5.0, 10.0)
	if err != nil {
		t.Fatalf("Failed to move relatively: %v", err)
	}
}

func TestVirtualPointerButtons(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer func() { _ = pointer.Close() }()

	// Test button press
	err = pointer.Button(time.Now(), BTN_LEFT, ButtonStatePressed)
	if err != nil {
		t.Fatalf("Failed to press button: %v", err)
	}

	// Test button release
	err = pointer.Button(time.Now(), BTN_LEFT, ButtonStateReleased)
	if err != nil {
		t.Fatalf("Failed to release button: %v", err)
	}

	// Test convenience methods
	err = pointer.LeftClick()
	if err != nil {
		t.Fatalf("Failed to perform left click: %v", err)
	}

	err = pointer.RightClick()
	if err != nil {
		t.Fatalf("Failed to perform right click: %v", err)
	}

	err = pointer.MiddleClick()
	if err != nil {
		t.Fatalf("Failed to perform middle click: %v", err)
	}
}

func TestVirtualPointerAxis(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer func() { _ = pointer.Close() }()

	// Test axis source
	err = pointer.AxisSource(AxisSourceWheel)
	if err != nil {
		t.Fatalf("Failed to set axis source: %v", err)
	}

	// Test axis event
	err = pointer.Axis(time.Now(), AxisVertical, 10.0)
	if err != nil {
		t.Fatalf("Failed to send axis event: %v", err)
	}

	// Test axis stop
	err = pointer.AxisStop(time.Now(), AxisVertical)
	if err != nil {
		t.Fatalf("Failed to send axis stop: %v", err)
	}

	// Test axis discrete
	err = pointer.AxisDiscrete(time.Now(), AxisVertical, 10.0, 1)
	if err != nil {
		t.Fatalf("Failed to send axis discrete: %v", err)
	}

	// Test convenience scroll methods
	err = pointer.ScrollVertical(10.0)
	if err != nil {
		t.Fatalf("Failed to scroll vertically: %v", err)
	}

	err = pointer.ScrollHorizontal(5.0)
	if err != nil {
		t.Fatalf("Failed to scroll horizontally: %v", err)
	}
}

func TestVirtualPointerFrame(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer func() { _ = pointer.Close() }()

	// Test frame
	err = pointer.Frame()
	if err != nil {
		t.Fatalf("Failed to send frame: %v", err)
	}
}

func TestVirtualPointerDestroy(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer func() { _ = manager.Close() }()

	pointer, err := manager.CreatePointer()
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}

	// Test destroy
	err = pointer.Close()
	if err != nil {
		t.Fatalf("Failed to close pointer: %v", err)
	}

	// Note: The API doesn't guarantee errors after Close() is called
	// so we don't test for that behavior
}

func TestButtonConstants(t *testing.T) {
	// Test that button constants are defined
	buttons := []uint32{BTN_LEFT, BTN_RIGHT, BTN_MIDDLE, BTN_SIDE, BTN_EXTRA}
	for _, button := range buttons {
		if button == 0 {
			t.Fatal("Button constant should not be zero")
		}
	}

	// Test button states
	if BUTTON_STATE_RELEASED != 0 {
		t.Fatal("BUTTON_STATE_RELEASED should be 0")
	}
	if BUTTON_STATE_PRESSED != 1 {
		t.Fatal("BUTTON_STATE_PRESSED should be 1")
	}
}

func TestAxisConstants(t *testing.T) {
	// Test axis constants
	if AXIS_VERTICAL_SCROLL != 0 {
		t.Fatal("AXIS_VERTICAL_SCROLL should be 0")
	}
	if AXIS_HORIZONTAL_SCROLL != 1 {
		t.Fatal("AXIS_HORIZONTAL_SCROLL should be 1")
	}

	// Test axis source constants
	sources := []uint32{AXIS_SOURCE_WHEEL, AXIS_SOURCE_FINGER, AXIS_SOURCE_CONTINUOUS, AXIS_SOURCE_WHEEL_TILT}
	for i, source := range sources {
		expected := uint32(i)
		if source != expected {
			t.Fatalf("Axis source constant %d should be %d, got %d", i, expected, source)
		}
	}
}

