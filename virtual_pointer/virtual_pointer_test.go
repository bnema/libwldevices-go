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

	// Test manager destruction
	err = manager.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy manager: %v", err)
	}
}

func TestVirtualPointerCreation(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer manager.Destroy()

	// Test creating virtual pointer without seat
	pointer, err := manager.CreateVirtualPointer(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	if pointer == nil {
		t.Fatal("Pointer should not be nil")
	}

	// Test creating virtual pointer with output
	pointer2, err := manager.CreateVirtualPointerWithOutput(nil, nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer with output: %v", err)
	}
	if pointer2 == nil {
		t.Fatal("Pointer2 should not be nil")
	}

	// Clean up
	pointer.Destroy()
	pointer2.Destroy()
}

func TestVirtualPointerMotion(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer manager.Destroy()

	pointer, err := manager.CreateVirtualPointer(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer pointer.Destroy()

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

	// Test invalid absolute motion (coordinates out of bounds)
	err = pointer.MotionAbsolute(time.Now(), 2000, 200, 1920, 1080)
	if err == nil {
		t.Fatal("Expected error for out of bounds coordinates")
	}
}

func TestVirtualPointerButtons(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer manager.Destroy()

	pointer, err := manager.CreateVirtualPointer(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer pointer.Destroy()

	// Test button press
	err = pointer.Button(time.Now(), BTN_LEFT, BUTTON_STATE_PRESSED)
	if err != nil {
		t.Fatalf("Failed to press button: %v", err)
	}

	// Test button release
	err = pointer.Button(time.Now(), BTN_LEFT, BUTTON_STATE_RELEASED)
	if err != nil {
		t.Fatalf("Failed to release button: %v", err)
	}

	// Test convenience methods
	err = pointer.ButtonPress(BTN_RIGHT)
	if err != nil {
		t.Fatalf("Failed to press button with convenience method: %v", err)
	}

	err = pointer.ButtonRelease(BTN_RIGHT)
	if err != nil {
		t.Fatalf("Failed to release button with convenience method: %v", err)
	}

	// Test invalid button state
	err = pointer.Button(time.Now(), BTN_LEFT, 999)
	if err == nil {
		t.Fatal("Expected error for invalid button state")
	}
}

func TestVirtualPointerAxis(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer manager.Destroy()

	pointer, err := manager.CreateVirtualPointer(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer pointer.Destroy()

	// Test axis source
	err = pointer.AxisSource(AXIS_SOURCE_WHEEL)
	if err != nil {
		t.Fatalf("Failed to set axis source: %v", err)
	}

	// Test axis event
	err = pointer.Axis(time.Now(), AXIS_VERTICAL_SCROLL, 10.0)
	if err != nil {
		t.Fatalf("Failed to send axis event: %v", err)
	}

	// Test axis stop
	err = pointer.AxisStop(time.Now(), AXIS_VERTICAL_SCROLL)
	if err != nil {
		t.Fatalf("Failed to send axis stop: %v", err)
	}

	// Test axis discrete
	err = pointer.AxisDiscrete(time.Now(), AXIS_VERTICAL_SCROLL, 10.0, 1)
	if err != nil {
		t.Fatalf("Failed to send axis discrete: %v", err)
	}

	// Test invalid axis
	err = pointer.Axis(time.Now(), 999, 10.0)
	if err == nil {
		t.Fatal("Expected error for invalid axis")
	}

	// Test invalid axis source
	err = pointer.AxisSource(999)
	if err == nil {
		t.Fatal("Expected error for invalid axis source")
	}
}

func TestVirtualPointerFrame(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer manager.Destroy()

	pointer, err := manager.CreateVirtualPointer(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer pointer.Destroy()

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
	defer manager.Destroy()

	pointer, err := manager.CreateVirtualPointer(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}

	// Test destroy
	err = pointer.Destroy()
	if err != nil {
		t.Fatalf("Failed to destroy pointer: %v", err)
	}

	// Test operations after destroy should fail
	err = pointer.Motion(time.Now(), 10.0, 20.0)
	if err == nil {
		t.Fatal("Expected error for operation on destroyed pointer")
	}
}

func TestConvenienceFunctions(t *testing.T) {
	ctx := context.Background()
	manager, err := NewVirtualPointerManager(ctx)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer manager: %v", err)
	}
	defer manager.Destroy()

	pointer, err := manager.CreateVirtualPointer(nil)
	if err != nil {
		t.Fatalf("Failed to create virtual pointer: %v", err)
	}
	defer pointer.Destroy()

	// Test click
	err = Click(pointer, BTN_LEFT)
	if err != nil {
		t.Fatalf("Failed to perform click: %v", err)
	}

	// Test scroll functions
	err = ScrollVertical(pointer, 10.0)
	if err != nil {
		t.Fatalf("Failed to perform vertical scroll: %v", err)
	}

	err = ScrollHorizontal(pointer, 5.0)
	if err != nil {
		t.Fatalf("Failed to perform horizontal scroll: %v", err)
	}

	// Test move functions
	err = MoveRelative(pointer, 10.0, 20.0)
	if err != nil {
		t.Fatalf("Failed to perform relative move: %v", err)
	}

	err = MoveAbsolute(pointer, 100, 200, 1920, 1080)
	if err != nil {
		t.Fatalf("Failed to perform absolute move: %v", err)
	}
}

func TestVirtualPointerError(t *testing.T) {
	err := &VirtualPointerError{
		Code:    ERROR_INVALID_AXIS,
		Message: "test error",
	}

	expected := "virtual pointer error 0: test error"
	if err.Error() != expected {
		t.Fatalf("Expected error message '%s', got '%s'", expected, err.Error())
	}
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
		if source != uint32(i) {
			t.Fatalf("Axis source constant %d should be %d, got %d", i, i, source)
		}
	}
}

func TestErrorConstants(t *testing.T) {
	if ERROR_INVALID_AXIS != 0 {
		t.Fatal("ERROR_INVALID_AXIS should be 0")
	}
	if ERROR_INVALID_AXIS_SOURCE != 1 {
		t.Fatal("ERROR_INVALID_AXIS_SOURCE should be 1")
	}
}